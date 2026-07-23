// uPlot 위젯 메타(Widgets/apiResponse/chartData)를 ECharts option으로 변환하는 빌더.
// 시리즈 순서·라벨은 uPlot 경로(prepareUPlotConfig)와 동일 규칙을 따른다.
import { Timezone } from 'components/CustomTimePicker/timezoneUtils';
import { getYAxisFormattedValue } from 'components/Graph/yAxisConfig';
import dayjs from 'dayjs';
import timezonePlugin from 'dayjs/plugin/timezone';
import utcPlugin from 'dayjs/plugin/utc';
import { getLegend } from 'lib/dashboard/getQueryResults';
import { convertValue } from 'lib/getConvertedValue';
import getLabelName from 'lib/getLabelName';
import { LineInterpolation, LineStyle } from 'lib/uPlotV2/config/types';
import { isInvalidPlotValue } from 'lib/uPlotV2/utils/dataUtils';
import type uPlot from 'uplot';
import { Widgets } from 'types/api/dashboard/getAll';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import { EChartsOption } from '../echartsCore';
import { getSeriesColor, MOCKUP_TUNING } from '../themes/dsapmTheme';

// 이 모듈에서 dayjs .tz()를 쓰므로 플러그인을 직접 확장한다 (repo 관행 —
// timeUtils.ts/useTimezoneFormatter.ts와 동일 패턴, side-effect import elision 회피)
dayjs.extend(utcPlugin);
dayjs.extend(timezonePlugin);

const SEC_TO_MS = 1000;

interface BuildArgs {
	widget: Widgets;
	apiResponse?: MetricRangePayloadProps;
	chartData: uPlot.AlignedData;
	currentQuery?: Query;
	isDarkMode: boolean;
	reducedMotion: boolean;
	minTimeScale?: number;
	maxTimeScale?: number;
	timezone?: Timezone;
	/** seriesIndex(uPlot 규약 1..n) → 표시 여부. false는 option에서 제외 */
	visibilityMap?: Record<number, boolean>;
}

interface SeriesDataPoint extends Array<number | null> {
	0: number;
	1: number | null;
}

interface LineSeriesItem {
	id: string;
	name: string;
	type: 'line';
	smooth: boolean;
	step?: 'end' | 'start';
	symbol: string;
	showSymbol: boolean;
	symbolSize: number;
	color: string;
	lineStyle: { width: number; type: 'dashed' | 'solid' };
	emphasis: { focus: 'series'; lineStyle: { width: number } };
	blur: { lineStyle: { opacity: number } };
	connectNulls: boolean;
	areaStyle?: {
		color: {
			type: 'linear';
			x: number;
			y: number;
			x2: number;
			y2: number;
			colorStops: Array<{ offset: number; color: string }>;
		};
	};
	data: SeriesDataPoint[];
	markLine?: Record<string, unknown>;
}

/** uPlot 경로(prepareUPlotConfig)와 동일한 규칙으로 시리즈 라벨을 만든다 */
function resolveSeriesLabels(
	apiResponse: MetricRangePayloadProps | undefined,
	currentQuery: Query | undefined,
): string[] {
	const result = apiResponse?.data?.result ?? [];
	return result.map((series) => {
		const base = getLabelName(
			series.metric,
			series.queryName || '',
			series.legend || '',
		);
		return currentQuery ? getLegend(series, currentQuery, base) : base;
	});
}

function buildMarkLine(widget: Widgets): Record<string, unknown> | undefined {
	const thresholds = widget.thresholds ?? [];
	const data = thresholds
		.map((t) => {
			if (t.thresholdValue === undefined) {
				return null;
			}
			const yAxis = convertValue(
				t.thresholdValue,
				t.thresholdUnit,
				widget.yAxisUnit,
			);
			if (yAxis === null) {
				return null;
			}
			return {
				yAxis,
				name: t.thresholdLabel ?? '',
				lineStyle: { color: t.thresholdColor ?? '#F0531C', type: 'dashed' },
				label: { formatter: t.thresholdLabel ?? '', position: 'insideEndTop' },
			};
		})
		.filter((d): d is NonNullable<typeof d> => d !== null);

	if (data.length === 0) {
		return undefined;
	}
	return { symbol: 'none', animation: false, data };
}

export function buildTimeSeriesOption({
	widget,
	apiResponse,
	chartData,
	currentQuery,
	isDarkMode,
	reducedMotion,
	minTimeScale,
	maxTimeScale,
	timezone,
	visibilityMap,
}: BuildArgs): { option: EChartsOption; seriesLabels: string[] } {
	const seriesLabels = resolveSeriesLabels(apiResponse, currentQuery);
	const timestamps = (chartData[0] ?? []) as number[];
	const markLine = buildMarkLine(widget);

	// 리뷰 반영: uPlot 경로(prepareUPlotConfig)가 반영하는 위젯 옵션과 패리티
	const interpolation = widget.lineInterpolation || LineInterpolation.Spline;
	const smooth = interpolation === LineInterpolation.Spline;
	let step: 'end' | 'start' | undefined;
	if (interpolation === LineInterpolation.StepAfter) {
		step = 'end';
	} else if (interpolation === LineInterpolation.StepBefore) {
		step = 'start';
	}

	const series: LineSeriesItem[] = seriesLabels
		.map((label, index): LineSeriesItem | null => {
			const seriesIndex = index + 1; // uPlot 규약 (0=x축)
			// 숨김 시리즈는 option에서 제외 — replaceMerge가 제거 처리(유령 없음).
			// 재표시 시 진입 애니메이션 재재생은 감수 (Task 12 확인 항목)
			if (visibilityMap && visibilityMap[seriesIndex] === false) {
				return null;
			}
			const color = getSeriesColor(
				label,
				widget.customLegendColors ?? {},
				isDarkMode,
			);
			const values = (chartData[seriesIndex] ?? []) as (number | null)[];
			// 유효 포인트 1개 시리즈는 라인이 안 그려져 사라지므로 심볼 강제
			// (uPlot 경로의 DrawStyle.Points 강제 대응)
			const validCount = values.filter((v) => !isInvalidPlotValue(v)).length;
			const showSymbol = Boolean(widget.showPoints) || validCount <= 1;
			return {
				// 라벨 중복 대비 인덱스 접두 유일 id — 리프레시 간 라벨+순서가
				// 안정하면 morphing 유지. name은 라벨 그대로(범례 액션 기준)
				id: `${index}:${label}`,
				name: label,
				type: 'line',
				smooth,
				step,
				symbol: 'circle',
				showSymbol,
				symbolSize: 5,
				color,
				lineStyle: {
					width: MOCKUP_TUNING.lineWidth,
					type: widget.lineStyle === LineStyle.Dashed ? 'dashed' : 'solid',
				},
				emphasis: {
					focus: 'series',
					lineStyle: { width: MOCKUP_TUNING.emphasisLineWidth },
				},
				blur: { lineStyle: { opacity: MOCKUP_TUNING.blurOpacity } },
				connectNulls: Boolean(widget.spanGaps ?? true),
				areaStyle: MOCKUP_TUNING.areaGradient
					? {
							color: {
								type: 'linear',
								x: 0,
								y: 0,
								x2: 0,
								y2: 1,
								colorStops: [
									{ offset: 0, color: `${color}${MOCKUP_TUNING.areaAlphaTop}` },
									{ offset: 1, color: `${color}00` },
								],
							},
					  }
					: undefined,
				data: timestamps.map(
					(ts, i) => [ts * SEC_TO_MS, values[i] ?? null] as SeriesDataPoint,
				),
			};
		})
		.filter((s): s is LineSeriesItem => s !== null);

	// 임계선은 첫 "표시" 시리즈에만 부착 (중복 렌더 방지 — 숨김 필터 이후 기준)
	if (series.length > 0 && markLine) {
		series[0].markLine = markLine;
	}

	const yAxisUnit = widget.yAxisUnit ?? 'none';
	const softMin = widget.softMin ?? undefined;
	const softMax = widget.softMax ?? undefined;

	const option = {
		animation: !reducedMotion,
		animationDuration: MOCKUP_TUNING.entryDurationMs,
		animationDurationUpdate: MOCKUP_TUNING.updateDurationMs,
		animationEasing: 'cubicOut',
		grid: { left: 8, right: 8, top: 12, bottom: 8, containLabel: true },
		// 범례 UI는 React Legend가 담당 — 시리즈 토글 액션용으로만 숨김 등록
		legend: { show: false, data: seriesLabels },
		xAxis: {
			type: 'time',
			min: minTimeScale !== undefined ? minTimeScale * SEC_TO_MS : undefined,
			max: maxTimeScale !== undefined ? maxTimeScale * SEC_TO_MS : undefined,
			axisLabel: {
				// 선택 timezone으로 축 라벨 표시 (스펙 §4 — TooltipHeader와 동일 규약).
				formatter: (value: number): string =>
					timezone
						? dayjs(value).tz(timezone.value).format('HH:mm:ss')
						: dayjs(value).format('HH:mm:ss'),
				hideOverlap: true,
			},
		},
		yAxis: {
			// 로그축 위젯 반영 — echarts log축은 0 이하 값 미지원. Task 12 실측에서
			// 0 포함 데이터 동작 확인, 문제 시 로그축 위젯의 uPlot 강등을 논의
			type: widget.isLogScale ? 'log' : 'value',
			scale: true,
			// softMin/softMax — uPlot soft 한계 대응(축이 최소 이 값까지 확장,
			// 데이터가 넘으면 데이터 우선). echarts는 min/max 콜백에 {min,max}
			// extent 전체를 넘긴다(브리프 정정 — 실타입은 min 콜백도 max를 받음)
			min:
				softMin !== undefined
					? (extent: { min: number; max: number }): number =>
							Math.min(extent.min, softMin)
					: undefined,
			max:
				softMax !== undefined
					? (extent: { min: number; max: number }): number =>
							Math.max(extent.max, softMax)
					: undefined,
			axisLabel: {
				// uPlot 경로와 동일한 단위 포맷 (components/Graph/yAxisConfig)
				formatter: (value: number): string =>
					getYAxisFormattedValue(String(value), yAxisUnit, widget.decimalPrecision),
			},
		},
		axisPointer: {
			show: true,
			snap: true,
			lineStyle: { color: isDarkMode ? '#5f6570' : '#9aa0a6' },
		},
		series,
	};

	// echarts의 실제 SeriesOption/XAXisOption 등은 등록된 컴포넌트 조합에 따라
	// 매우 큰 유니온 타입이라 인라인 리터럴을 바로 EChartsOption으로 선언하면
	// (브리프 원안처럼) 불필요한 초과 프로퍼티 검사에 걸린다. 여기서 한 번만
	// 캐스팅해 계약(반환 타입)은 유지하면서 내부 구현은 구조적으로 느슨하게 둔다.
	return { option: option as EChartsOption, seriesLabels };
}
