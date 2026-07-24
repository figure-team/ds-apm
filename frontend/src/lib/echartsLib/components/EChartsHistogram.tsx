// 히스토그램 전용 경량 조립. 값 x축·HistogramTooltip·호버+핀만 있어
// EChartsCartesian(시간축·클릭·드래그)을 쓰지 않는다. hover/핀 상태기계는
// useEChartsHoverPin으로 2a와 공유하고 버킷 인덱스 해석만 주입한다(스펙 R2).
// ⚠️ react import는 반드시 한 줄로 유지 (중복 임포트는 import/no-duplicates 린트 실패)
import { useCallback, useMemo, useState } from 'react';
import { PrecisionOption } from 'components/Graph/types';
import ChartLayout from 'container/DashboardContainer/visualization/layout/ChartLayout/ChartLayout';
import Legend from 'lib/uPlotV2/components/Legend/Legend';
import HistogramTooltip from 'lib/uPlotV2/components/Tooltip/HistogramTooltip';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import type uPlot from 'uplot';
import { Widgets } from 'types/api/dashboard/getAll';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import {
	buildHistogramOption,
	MERGED_SERIES_COLOR,
} from '../builders/histogramOption';
import EChartsPlotContextProvider from '../context/EChartsPlotContextProvider';
import {
	AxisPointerEventInfo,
	useEChartsHoverPin,
} from '../hooks/useEChartsHoverPin';
import { getSeriesColor } from '../themes/dsapmTheme';
import { resolveBucketIndex } from '../utils/histogramBuckets';
import { buildUPlotShim } from '../utils/uplotShim';
import EChartsTooltipPositioner from './EChartsTooltipPositioner';
import EChartsView from './EChartsView';

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';

export interface EChartsHistogramProps {
	widget: Widgets;
	chartData: uPlot.AlignedData;
	/** 범례 계약용 병행 config — Legend/ChartLayout이 소비 */
	configBuilder: UPlotConfigBuilder;
	apiResponse?: MetricRangePayloadProps;
	currentQuery?: Query;
	isDarkMode: boolean;
	yAxisUnit?: string;
	decimalPrecision?: PrecisionOption;
	legendPosition: LegendPosition;
	/** widget.mergeAllActiveQueries — 병합 시 범례 숨김 + 단일 시리즈 */
	isQueriesMerged: boolean;
	canPinTooltip?: boolean;
	width: number;
	height: number;
	layoutChildren?: React.ReactNode;
	children?: React.ReactNode;
	onEngineError: (error: unknown) => void;
}

export default function EChartsHistogram({
	widget,
	chartData,
	configBuilder,
	apiResponse,
	currentQuery,
	isDarkMode,
	yAxisUnit,
	decimalPrecision,
	legendPosition,
	isQueriesMerged,
	canPinTooltip = false,
	width,
	height,
	layoutChildren,
	children,
	onEngineError,
}: EChartsHistogramProps): JSX.Element {
	const [visibilityMap, setVisibilityMap] = useState<Record<number, boolean>>({});

	const reducedMotion = useMemo(
		() =>
			typeof window !== 'undefined' &&
			typeof window.matchMedia === 'function' &&
			window.matchMedia(REDUCED_MOTION_QUERY).matches,
		[],
	);

	const { option, seriesLabels } = useMemo(
		() =>
			buildHistogramOption({
				widget,
				apiResponse,
				chartData,
				currentQuery,
				isDarkMode,
				reducedMotion,
				isQueriesMerged,
				visibilityMap,
			}),
		[
			widget,
			apiResponse,
			chartData,
			currentQuery,
			isDarkMode,
			reducedMotion,
			isQueriesMerged,
			visibilityMap,
		],
	);

	// 히스토그램: axisPointer x값 → 버킷 bracket 인덱스. 유령 널빈은 스킵(R2·R3)
	const resolveIndex = useCallback(
		(info: AxisPointerEventInfo): number | null => {
			const xAxisInfo = info.axesInfo?.find((a) => a.axisDim === 'x');
			if (!xAxisInfo || typeof xAxisInfo.value !== 'number') {
				return null;
			}
			return resolveBucketIndex(
				xAxisInfo.value,
				(chartData[0] ?? []) as number[],
				{ skipLeadingNullBin: true },
			);
		},
		[chartData],
	);

	const { chart, hover, mousePos, dismissTooltip, handleInstanceReady } =
		useEChartsHoverPin({ canPinTooltip, resolveIndex });

	const seriesMeta = useMemo(
		() =>
			seriesLabels.map((label, i) => ({
				label,
				// 병합 모드는 label===''이라 getSeriesColor가 해시 팔레트색을 내는데,
				// 막대는 MERGED_SERIES_COLOR로 그려지므로 스와치도 같은 상수를 써야
				// 색이 일치한다(N-M1). 빌더(builders/histogramOption.ts)와 동일 상수 재사용.
				color: isQueriesMerged
					? MERGED_SERIES_COLOR
					: getSeriesColor(label, widget.customLegendColors ?? {}, isDarkMode),
				show: visibilityMap[i + 1] ?? true,
			})),
		[seriesLabels, widget.customLegendColors, isDarkMode, visibilityMap, isQueriesMerged],
	);

	// HistogramTooltip 재사용을 위한 uPlot 심 — cursor.idx에 버킷 인덱스를 주입
	const shim = useMemo(
		() => buildUPlotShim(chartData, seriesMeta, hover.dataIndex),
		[chartData, seriesMeta, hover.dataIndex],
	);

	// I-1: "표시 시리즈가 1개"가 아니라 "해당 버킷에서 유한값을 가진(=기여하는)
	// 표시 시리즈가 1개"를 봐야 한다. dataIndexes는 모든 시리즈에 같은 버킷 행을
	// 주지만, buildTooltipContent(uPlotV2/Tooltip/utils.ts)는 그 행 값이 유한하지
	// 않은(null) 시리즈를 content에서 통째로 제외한다. 그래서 "표시 2개 이상 ·
	// 그 버킷에서 값 있는 시리즈는 1개"인 흔한 케이스(mergeAlignedDataTables가
	// 버킷 합집합으로 정렬해 꼬리 구간에서 상시 발생)에서 activeSeriesIndex가
	// 계속 null로 남아 activeItem이 안 잡히고, HistogramTooltip은 헤더가 항상
	// 꺼져 있어(showTooltipHeader=false) 라벨·카운트가 어디에도 안 뜨는 빈
	// 툴팁이 된다. 기여 시리즈 기준으로 좁히면 그 1개가 activeItem이 되어
	// 헤더에 라벨+카운트가 표시된다. 기여가 2개 이상이면 기존대로 null
	// (TooltipList가 값을 나열).
	const tooltipSeriesIndex = useMemo(() => {
		if (hover.dataIndex === null) {
			return null;
		}
		const { dataIndex } = hover;
		let found: number | null = null;
		for (let i = 0; i < seriesMeta.length; i += 1) {
			if (!seriesMeta[i].show) {
				continue;
			}
			const column = chartData[i + 1] as (number | null)[] | undefined;
			const value = column?.[dataIndex];
			if (!Number.isFinite(value)) {
				continue;
			}
			if (found !== null) {
				return null;
			}
			found = i + 1;
		}
		return found;
	}, [seriesMeta, chartData, hover.dataIndex]);

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
				showLegend={!isQueriesMerged}
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
							data-testid="echarts-histogram-view"
						/>
						{hover.dataIndex !== null && (
							<EChartsTooltipPositioner position={mousePos} isPinned={hover.pinned}>
								<HistogramTooltip
									uPlotInstance={shim}
									dataIndexes={[null, ...seriesLabels.map(() => hover.dataIndex)]}
									seriesIndex={tooltipSeriesIndex}
									isPinned={hover.pinned}
									dismiss={dismissTooltip}
									viaSync={false}
									yAxisUnit={yAxisUnit}
									decimalPrecision={decimalPrecision}
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
