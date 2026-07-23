import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Timezone } from 'components/CustomTimePicker/timezoneUtils';
import { PrecisionOption } from 'components/Graph/types';
import ChartLayout from 'container/DashboardContainer/visualization/layout/ChartLayout/ChartLayout';
import { OnClickPluginOpts } from 'lib/uPlotLib/plugins/onClickPlugin';
import Legend from 'lib/uPlotV2/components/Legend/Legend';
import TimeSeriesTooltip from 'lib/uPlotV2/components/Tooltip/TimeSeriesTooltip';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import type uPlot from 'uplot';
import { Widgets } from 'types/api/dashboard/getAll';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import EChartsPlotContextProvider from '../context/EChartsPlotContextProvider';
import { EChartsOption, EChartsType } from '../echartsCore';
import {
	DRAG_CLICK_DIST_PX,
	EChartsClickInfo,
	useEChartsEvents,
} from '../hooks/useEChartsEvents';
import { getSeriesColor } from '../themes/dsapmTheme';
import { buildUPlotShim } from '../utils/uplotShim';
import EChartsTooltipPositioner from './EChartsTooltipPositioner';
import EChartsView from './EChartsView';

export interface EChartsCartesianProps {
	widget: Widgets;
	chartData: uPlot.AlignedData;
	/** 범례 계약용 병행 config (스펙 §3.3) — Legend/ChartLayout이 실제 소비 */
	configBuilder: UPlotConfigBuilder;
	apiResponse?: MetricRangePayloadProps;
	currentQuery?: Query;
	isDarkMode: boolean;
	timezone: Timezone;
	yAxisUnit?: string;
	decimalPrecision?: PrecisionOption;
	legendPosition: LegendPosition;
	minTimeScale?: number;
	maxTimeScale?: number;
	onDragSelect: (start: number, end: number) => void;
	onClick?: OnClickPluginOpts['onClick'];
	canPinTooltip?: boolean;
	width: number;
	height: number;
	layoutChildren?: React.ReactNode;
	children?: React.ReactNode;
	/** TimeSeriesPanel의 폴백 트리거 */
	onEngineError: (error: unknown) => void;
	/** 가변점: 조립 내부 상태로 option을 생성 (스펙 §3.2 seam) */
	buildOption: (ctx: {
		visibilityMap: Record<number, boolean>;
		reducedMotion: boolean;
	}) => { option: EChartsOption; seriesLabels: string[] };
	/** 클릭 시리즈 특정 방식 — 'line'=zrender 픽셀+y최근접, 'bar'=시리즈 click 이벤트 (스펙 §4.1) */
	clickMode: 'line' | 'bar';
}

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';
// uPlot 경로 패리티: 핀은 호버 중 'p' 키 (DEFAULT_PIN_TOOLTIP_KEY — 클릭은 메뉴 전용)
const PIN_TOOLTIP_KEY = 'p';
const MS_PER_SEC = 1000;

interface HoverState {
	dataIndex: number | null;
	pinned: boolean;
}

/** timestamps(초, 오름차순)에서 targetSec에 가장 가까운 인덱스 */
function nearestIndex(timestamps: number[], targetSec: number): number {
	let best = 0;
	let bestDiff = Infinity;
	for (let i = 0; i < timestamps.length; i += 1) {
		const diff = Math.abs(timestamps[i] - targetSec);
		if (diff < bestDiff) {
			bestDiff = diff;
			best = i;
		}
	}
	return best;
}

interface FocusedSeries {
	seriesIndex: number;
	seriesName: string;
	value: number;
	color: string;
	show: boolean;
	isFocused: boolean;
}

/** 클릭 y값 최근접 "표시" 시리즈 특정 — onClickPlugin.getFocusedSeriesAtPosition 대응 */
function resolveFocusedSeries(
	chartData: uPlot.AlignedData,
	seriesMeta: Array<{ label: string; color: string; show: boolean }>,
	dataIndex: number,
	yValue: number | null,
): FocusedSeries | null {
	let best: FocusedSeries | null = null;
	let bestDiff = Infinity;
	seriesMeta.forEach((meta, i) => {
		if (!meta.show) {
			return;
		}
		const value = (chartData[i + 1] as (number | null)[] | undefined)?.[dataIndex];
		if (value == null || Number.isNaN(value)) {
			return;
		}
		const diff = yValue === null ? 0 : Math.abs(value - yValue);
		if (diff < bestDiff) {
			bestDiff = diff;
			best = {
				seriesIndex: i + 1,
				seriesName: meta.label,
				value,
				color: meta.color,
				show: true,
				isFocused: true,
			};
		}
	});
	return best;
}

/**
 * 표시 중인 시리즈가 정확히 1개면 그 uPlot 인덱스(1..n)를 반환한다(리뷰 Critical #1).
 *
 * Tooltip.tsx 렌더 규칙: `showList = content.length > 1`이라 시리즈가 1개면
 * TooltipList가 렌더되지 않고, activeItem(`content.find(isActive)`)만으로 값을
 * 표현해야 한다. activeItem은 `seriesIndex===activeSeriesIndex` 매칭이 필요한데
 * seriesIndex를 항상 null로 고정하면 단일 시리즈 패널에서 헤더의 타임스탬프만
 * 뜨고 값이 어디에도 표시되지 않는다. 표시 시리즈가 2개 이상이면 TooltipList가
 * 값을 표현하므로 null을 유지한다(uPlot의 포커스 하이라이트 부재는 허용된 패리티 갭).
 */
function resolveSingleVisibleSeriesIndex(
	seriesMeta: Array<{ show: boolean }>,
): number | null {
	let found: number | null = null;
	for (let i = 0; i < seriesMeta.length; i += 1) {
		if (seriesMeta[i].show) {
			if (found !== null) {
				return null; // 표시 시리즈 2개 이상 — 리스트 렌더로 이미 값 표시됨
			}
			found = i + 1;
		}
	}
	return found;
}

/** echarts 'updateAxisPointer' 이벤트 페이로드 — axisTrigger.js의 outputPayload 형태 */
interface AxisPointerEventInfo {
	dataIndex?: number;
	axesInfo?: Array<{ axisDim: string; value?: number }>;
}

/**
 * updateAxisPointer 이벤트에서 hover dataIndex를 추출한다.
 *
 * 조사 결과(echarts 소스 axisTrigger.js 확인): 이 이벤트의 실제 페이로드는
 * `{ axesInfo: [{ axisDim, axisIndex, value }] }` 형태이며 top-level dataIndex는
 * 없다(dataIndex는 'showTip' 액션 페이로드에만 존재). 따라서 axesInfo에서 x축
 * value(ms)를 찾아 timestamps(초)로 나눈 뒤 nearestIndex로 대체 계산한다.
 * 향후 echarts 버전이 top-level dataIndex를 제공하면 그 값을 우선한다.
 */
function resolveHoverDataIndex(
	info: AxisPointerEventInfo,
	timestamps: number[],
): number | null {
	if (typeof info.dataIndex === 'number') {
		return info.dataIndex;
	}
	const xAxisInfo = info.axesInfo?.find((a) => a.axisDim === 'x');
	if (
		xAxisInfo &&
		typeof xAxisInfo.value === 'number' &&
		timestamps.length > 0
	) {
		return nearestIndex(timestamps, xAxisInfo.value / MS_PER_SEC);
	}
	return null;
}

export default function EChartsCartesian({
	widget,
	chartData,
	configBuilder,
	apiResponse,
	currentQuery,
	isDarkMode,
	timezone,
	yAxisUnit,
	decimalPrecision,
	legendPosition,
	minTimeScale,
	maxTimeScale,
	onDragSelect,
	onClick,
	canPinTooltip = false,
	width,
	height,
	layoutChildren,
	children,
	onEngineError,
	buildOption,
	clickMode,
}: EChartsCartesianProps): JSX.Element {
	const [chart, setChart] = useState<EChartsType | null>(null);
	const [hover, setHover] = useState<HoverState>({
		dataIndex: null,
		pinned: false,
	});
	// 리뷰 반영: 표시 상태(단일 소스는 Provider) — option 재빌드·심 show·클릭 특정에 사용
	const [visibilityMap, setVisibilityMap] = useState<Record<number, boolean>>({});
	// 리뷰 반영: 툴팁 배치용 마우스 좌표 (EChartsTooltipPositioner 입력)
	const [mousePos, setMousePos] = useState<{
		clientX: number;
		clientY: number;
	} | null>(null);

	// handleInstanceReady는 인스턴스 생성 시 한 번만 호출되므로(EChartsView init
	// effect가 isDarkMode에만 의존) 그 안의 리스너가 최신 chartData를 읽으려면
	// ref가 필요하다(클로저 고정 방지)
	const chartDataRef = useRef(chartData);
	chartDataRef.current = chartData;

	// mousemove는 고빈도라 매 이벤트 setMousePos 시 리렌더가 잦다. rAF로 프레임당
	// 1회만 반영하고, 핀 상태에선 Positioner가 좌표를 무시하므로 갱신 자체를 건너뛴다.
	// 리스너는 handleInstanceReady 클로저에 고정되므로 최신 핀 상태는 ref로 읽는다.
	const pinnedRef = useRef(false);
	pinnedRef.current = hover.pinned;
	const mouseRafRef = useRef<number | null>(null);
	const pendingMouseRef = useRef<{ clientX: number; clientY: number } | null>(
		null,
	);

	const reducedMotion = useMemo(
		() =>
			typeof window !== 'undefined' &&
			typeof window.matchMedia === 'function' &&
			window.matchMedia(REDUCED_MOTION_QUERY).matches,
		[],
	);

	const { option, seriesLabels } = useMemo(
		() => buildOption({ visibilityMap, reducedMotion }),
		[buildOption, visibilityMap, reducedMotion],
	);

	// 시리즈 메타(라벨·색·표시) — 심과 클릭 시리즈 특정이 공유 (리뷰 반영)
	const seriesMeta = useMemo(
		() =>
			seriesLabels.map((label, i) => ({
				label,
				color: getSeriesColor(label, widget.customLegendColors ?? {}, isDarkMode),
				show: visibilityMap[i + 1] ?? true,
			})),
		[seriesLabels, widget.customLegendColors, isDarkMode, visibilityMap],
	);

	// 리뷰 반영 — 클릭은 10-인자 OnClickPluginOpts 계약으로 재구성해야 컨텍스트
	// 메뉴가 열린다(usePanelContextMenu는 queryData.queryName 없으면 메뉴를 안 연다)
	const handleClick = useCallback(
		(click: EChartsClickInfo): void => {
			if (!onClick || click.xValueMs === null) {
				return;
			}
			const xValueSec = click.xValueMs / MS_PER_SEC; // uPlot 규약: 초
			const timestamps = (chartData[0] ?? []) as number[];
			const dataIndex = nearestIndex(timestamps, xValueSec);
			const focused = resolveFocusedSeries(
				chartData,
				seriesMeta,
				dataIndex,
				click.yValue,
			);
			const result = apiResponse?.data?.result ?? [];
			const focusedApi = focused ? result[focused.seriesIndex - 1] : undefined;
			const metric = {
				...(focusedApi?.metric ?? {}),
				clickedTimestamp: timestamps[dataIndex] ?? xValueSec,
			};
			const queryData = {
				queryName: focusedApi?.queryName ?? '',
				inFocusOrNot: Boolean(focusedApi),
			};
			onClick(
				xValueSec,
				click.yValue ?? 0,
				click.mouseX + 40, // uPlot 경로 보정치 패리티 (onClickPlugin.ts:143-144)
				click.mouseY + 40,
				metric as never,
				queryData,
				click.absoluteMouseX,
				click.absoluteMouseY,
				// axesData — usePanelContextMenu의 timeRange 계산 게이트용 truthy
				{ xAxis: {}, yAxis: {} } as never,
				focused,
			);
		},
		[onClick, chartData, seriesMeta, apiResponse],
	);
	// 라인: 기존 zrender 픽셀 클릭 경로 유지. 막대: zrender 클릭 배선 끄고 시리즈 click 사용
	useEChartsEvents({
		chart,
		onDragSelect,
		onClick: clickMode === 'line' ? handleClick : undefined,
	});

	// 막대 클릭 — 시리즈 click 이벤트로 seriesId 직결 (스펙 §4.1).
	// ⚠️ 드래그 줌 충돌 방지: native ECharts click은 useEChartsEvents의 10px
	// 게이트(DRAG_CLICK_DIST_PX, 회귀 2242cbc4)를 안 거친다. onDragSelect는 막대
	// 모드에서도 zrender에 붙어 있으므로, 게이트 없이 두면 막대 위에서 드래그 줌
	// 직후 컨텍스트 메뉴가 함께 열릴 수 있다. zrender mousedown 위치를 기억했다가
	// 클릭 지점이 10px↑ 이동했으면(=드래그 제스처) 클릭을 억제해 라인 경로와 계약을 맞춘다.
	const barDownRef = useRef<{ x: number; y: number } | null>(null);
	const handleBarClick = useCallback(
		(params: {
			seriesId?: string;
			dataIndex?: number;
			event?: { event?: MouseEvent; offsetX?: number; offsetY?: number };
		}): void => {
			if (!onClick || params.seriesId === undefined || params.dataIndex === undefined) {
				return;
			}
			// 드래그 줌 제스처면 클릭 억제 (10px 계약 패리티)
			const down = barDownRef.current;
			const upX = params.event?.offsetX ?? 0;
			if (down && Math.abs(upX - down.x) >= DRAG_CLICK_DIST_PX) {
				return;
			}
			const originalIndex = Number(params.seriesId.split(':')[0]); // 안정 id `${index}:${label}` — 0-based
			if (Number.isNaN(originalIndex)) {
				return;
			}
			const seriesIndex = originalIndex + 1; // uPlot 규약
			const timestamps = (chartData[0] ?? []) as number[];
			const di = params.dataIndex;
			const tsSec = timestamps[di] ?? 0;
			const value =
				(chartData[seriesIndex] as (number | null)[] | undefined)?.[di] ?? 0;
			const result = apiResponse?.data?.result ?? [];
			const focusedApi = result[originalIndex];
			const focused = {
				seriesIndex,
				seriesName: seriesMeta[originalIndex]?.label ?? '',
				value: value ?? 0,
				color: seriesMeta[originalIndex]?.color ?? '',
				show: true,
				isFocused: true,
			};
			const native = params.event?.event;
			const mouseX = params.event?.offsetX ?? 0;
			const mouseY = params.event?.offsetY ?? 0;
			onClick(
				tsSec,
				value ?? 0,
				mouseX + 40, // uPlot 경로 보정치 패리티
				mouseY + 40,
				{ ...(focusedApi?.metric ?? {}), clickedTimestamp: tsSec } as never,
				{ queryName: focusedApi?.queryName ?? '', inFocusOrNot: Boolean(focusedApi) },
				native?.clientX ?? mouseX,
				native?.clientY ?? mouseY,
				{ xAxis: {}, yAxis: {} } as never,
				focused,
			);
		},
		[onClick, chartData, seriesMeta, apiResponse],
	);

	// 막대 모드일 때만 시리즈 click + mousedown(드래그 게이트) 등록
	// (stale 클로저 방지 — 별도 effect로 off/on)
	useEffect(() => {
		if (!chart || clickMode !== 'bar') {
			return undefined;
		}
		const zr = chart.getZr();
		const onDown = (e: { offsetX: number; offsetY: number }): void => {
			barDownRef.current = { x: e.offsetX, y: e.offsetY };
		};
		const handler = (params: unknown): void =>
			handleBarClick(params as Parameters<typeof handleBarClick>[0]);
		zr.on('mousedown', onDown);
		chart.on('click', handler);
		return (): void => {
			zr.off('mousedown', onDown);
			chart.off('click', handler);
		};
	}, [chart, clickMode, handleBarClick]);

	const dismissTooltip = useCallback(
		(): void => setHover({ dataIndex: null, pinned: false }),
		[],
	);

	// 리뷰 반영 — 핀은 uPlot 경로 패리티대로 호버 중 'p' 키(클릭은 메뉴 전용).
	// Esc는 재사용 중인 TooltipFooter(uPlot 경로 컴포넌트, 무수정 재사용)가
	// "Press P or Esc to unpin" 힌트를 그대로 렌더하므로 해제 전용으로 패리티 유지.
	useEffect(() => {
		if (!canPinTooltip) {
			return undefined;
		}
		const onKeyDown = (e: KeyboardEvent): void => {
			if (e.key === 'Escape') {
				setHover((prev) => (prev.pinned ? { ...prev, pinned: false } : prev));
				return;
			}
			// uPlot 경로 패리티(TooltipPlugin.tsx:310) — 대소문자 무시 비교
			if (e.key.toLowerCase() !== PIN_TOOLTIP_KEY) {
				return;
			}
			setHover((prev) =>
				prev.dataIndex === null && !prev.pinned
					? prev
					: { ...prev, pinned: !prev.pinned },
			);
		};
		window.addEventListener('keydown', onKeyDown);
		return (): void => window.removeEventListener('keydown', onKeyDown);
	}, [canPinTooltip]);

	// 툴팁 상태: axisPointer 이벤트 → dataIndex 추적 (핀 상태에선 갱신 정지)
	const handleInstanceReady = useCallback((instance: EChartsType): void => {
		setChart(instance);
		instance.on('updateAxisPointer', (e: unknown): void => {
			const info = e as AxisPointerEventInfo;
			const timestamps = (chartDataRef.current[0] ?? []) as number[];
			const dataIndex = resolveHoverDataIndex(info, timestamps);
			setHover((prev) => (prev.pinned ? prev : { ...prev, dataIndex }));
		});
		// 툴팁 배치용 마우스 추적 — rAF로 프레임당 1회만 반영(고빈도 리렌더 방지).
		// 핀 상태에선 Positioner가 좌표를 무시하므로 갱신 자체를 건너뛴다.
		instance.getZr().on('mousemove', (e: { event?: MouseEvent }): void => {
			const native = e.event;
			if (!native || pinnedRef.current) {
				return;
			}
			pendingMouseRef.current = {
				clientX: native.clientX,
				clientY: native.clientY,
			};
			if (mouseRafRef.current !== null) {
				return;
			}
			mouseRafRef.current = requestAnimationFrame(() => {
				mouseRafRef.current = null;
				if (pendingMouseRef.current) {
					setMousePos(pendingMouseRef.current);
				}
			});
		});
		instance.getZr().on('globalout', (): void => {
			setHover((prev) => (prev.pinned ? prev : { ...prev, dataIndex: null }));
		});
	}, []);

	// 언마운트 시 대기 중인 mousemove rAF 취소 (해제 후 setState 방지)
	useEffect(
		() => (): void => {
			if (mouseRafRef.current !== null) {
				cancelAnimationFrame(mouseRafRef.current);
				mouseRafRef.current = null;
			}
		},
		[],
	);

	// TimeSeriesTooltip 재사용을 위한 uPlot 심 (스펙 §4 툴팁 결정)
	// cursor.idx가 타임스탬프 헤더의 출처이므로 hover.dataIndex를 주입한다 (리뷰 반영)
	const shim = useMemo(
		() => buildUPlotShim(chartData, seriesMeta, hover.dataIndex),
		[chartData, seriesMeta, hover.dataIndex],
	);

	// 단일 시리즈 패널에서 activeItem으로 값이 표시되도록 배선 (리뷰 Critical #1)
	const tooltipSeriesIndex = useMemo(
		() => resolveSingleVisibleSeriesIndex(seriesMeta),
		[seriesMeta],
	);

	const legendComponent = useCallback(
		(averageLegendWidth: number): React.ReactNode => (
			<Legend
				config={configBuilder}
				position={legendPosition}
				averageLegendWidth={averageLegendWidth}
			/>
		),
		[configBuilder, legendPosition],
	);

	return (
		<EChartsPlotContextProvider
			chart={chart}
			widgetId={widget.id}
			seriesLabels={seriesLabels}
			config={configBuilder}
			onVisibilityChange={setVisibilityMap}
			shouldSaveSelectionPreference
		>
			<ChartLayout
				showLegend
				config={configBuilder}
				containerWidth={width}
				containerHeight={height}
				legendConfig={{ position: legendPosition }}
				legendComponent={legendComponent}
				layoutChildren={layoutChildren}
			>
				{({ chartWidth, chartHeight }): JSX.Element => (
					<div style={{ position: 'relative' }}>
						<EChartsView
							option={option}
							width={chartWidth}
							height={chartHeight}
							isDarkMode={isDarkMode}
							onError={onEngineError}
							onInstanceReady={handleInstanceReady}
							data-testid="echarts-cartesian"
						/>
						{hover.dataIndex !== null && (
							<EChartsTooltipPositioner position={mousePos} isPinned={hover.pinned}>
								<TimeSeriesTooltip
									uPlotInstance={shim}
									// 규약(리뷰): 길이 n+1, [0]=x축 자리 — buildTooltipContent가 1..n을 읽는다
									dataIndexes={[null, ...seriesLabels.map(() => hover.dataIndex)]}
									seriesIndex={tooltipSeriesIndex}
									isPinned={hover.pinned}
									dismiss={dismissTooltip}
									viaSync={false}
									yAxisUnit={yAxisUnit}
									decimalPrecision={decimalPrecision}
									timezone={timezone}
									canPinTooltip={canPinTooltip}
								/>
							</EChartsTooltipPositioner>
						)}
						{children}
					</div>
				)}
			</ChartLayout>
		</EChartsPlotContextProvider>
	);
}
