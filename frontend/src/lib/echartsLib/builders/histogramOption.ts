// 히스토그램 위젯 메타를 ECharts option으로 변환한다. 막대(barOption)의 사촌이지만
// x축이 시간이 아니라 버킷 값이고, 빈은 간격 없이 버킷 폭을 채운다(스펙 R1).
//
// R1 구현 주의(리뷰 2026-07-24): barWidth를 숫자로 주면 안 된다 — ECharts에서
// 숫자 barWidth는 데이터 단위가 아니라 픽셀이다(barGrid.js:177). value축의
// bandWidth는 이미 "인접 데이터 최소 간격(=bucketSize)의 픽셀 폭"이므로
// (barGrid.js:165-173), barCategoryGap/barGap을 0%로 두기만 하면
// autoWidth = bandWidth / N 이 되어 한 버킷을 N개 시리즈가 정확히 채운다
// (barGrid.js:258). 시리즈를 숨겨 N이 줄면 ECharts가 알아서 재계산한다.
// barCategoryGap은 value축에서도 적용되며, 미지정 시 기본값 max(35-4N,15)%가
// 버킷 사이에 갭을 만들어 버리므로 반드시 명시한다.
import { getYAxisFormattedValue } from 'components/Graph/yAxisConfig';
import type uPlot from 'uplot';
import { Widgets } from 'types/api/dashboard/getAll';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import { EChartsOption } from '../echartsCore';
import { BAR_MOCKUP_TUNING, getSeriesColor } from '../themes/dsapmTheme';
import { deriveBucketSize } from '../utils/histogramBuckets';
import { buildMarkLine, resolveSeriesLabels } from './timeSeriesOption';

/**
 * 병합 모드 색 — uPlot 경로(prepareHistogramPanelConfig)의 fillColor와 동일.
 * export: EChartsHistogram의 툴팁 스와치도 같은 상수를 써야 막대 색과 일치한다(N-M1).
 */
export const MERGED_SERIES_COLOR = '#D81B2C';

interface BuildArgs {
	widget: Widgets;
	apiResponse?: MetricRangePayloadProps;
	chartData: uPlot.AlignedData;
	currentQuery?: Query;
	isDarkMode: boolean;
	reducedMotion: boolean;
	/** widget.mergeAllActiveQueries — 병합 시 단일 시리즈(라벨 없음) */
	isQueriesMerged: boolean;
	/** seriesIndex(uPlot 규약 1..n) → 표시 여부. false는 option에서 제외 */
	visibilityMap?: Record<number, boolean>;
}

interface HistogramSeriesItem {
	id: string;
	name: string;
	type: 'bar';
	/** 시리즈 간 갭 0 — 한 버킷 안에서 나란히 붙는다 */
	barGap: string;
	/** 버킷 간 갭 0 — 그룹이 버킷 폭을 꽉 채운다(기본값은 15%↑ 갭) */
	barCategoryGap: string;
	itemStyle: {
		borderRadius: number[];
		color: {
			type: 'linear';
			x: number;
			y: number;
			x2: number;
			y2: number;
			colorStops: Array<{ offset: number; color: string }>;
		};
	};
	emphasis: { focus: 'series' };
	blur: { itemStyle: { opacity: number } };
	animationDelay: (idx: number) => number;
	data: Array<[number, number | null]>;
	markLine?: Record<string, unknown>;
}

function gradientOf(color: string): HistogramSeriesItem['itemStyle']['color'] {
	return {
		type: 'linear',
		x: 0,
		y: 0,
		x2: 0,
		y2: 1,
		colorStops: [
			{ offset: 0, color: `${color}${BAR_MOCKUP_TUNING.areaAlphaTop}` },
			{ offset: 1, color: `${color}${BAR_MOCKUP_TUNING.areaAlphaBottom}` },
		],
	};
}

export function buildHistogramOption({
	widget,
	apiResponse,
	chartData,
	currentQuery,
	isDarkMode,
	reducedMotion,
	isQueriesMerged,
	visibilityMap,
}: BuildArgs): { option: EChartsOption; seriesLabels: string[] } {
	const edges = (chartData[0] ?? []) as number[];
	const bucketSize = deriveBucketSize(edges);
	const markLine = buildMarkLine(widget);

	// 병합 모드는 데이터 컬럼이 1개뿐이라 라벨도 단일(빈 문자열 — uPlot 경로 패리티)
	const seriesLabels = isQueriesMerged
		? ['']
		: resolveSeriesLabels(apiResponse, currentQuery);

	const r = BAR_MOCKUP_TUNING.borderRadius;
	const half = bucketSize / 2;

	const series: HistogramSeriesItem[] = seriesLabels
		.map((label, index): HistogramSeriesItem | null => {
			const seriesIndex = index + 1; // uPlot 규약
			if (visibilityMap && visibilityMap[seriesIndex] === false) {
				return null;
			}
			const color = isQueriesMerged
				? MERGED_SERIES_COLOR
				: getSeriesColor(label, widget.customLegendColors ?? {}, isDarkMode);
			const counts = (chartData[seriesIndex] ?? []) as (number | null)[];
			return {
				id: `${index}:${label}`,
				name: label,
				type: 'bar',
				// R1 — 폭은 ECharts가 bandWidth(=버킷 폭 px)를 N등분해 정한다.
				// barWidth를 숫자로 주면 픽셀로 해석되므로 절대 넣지 않는다.
				barGap: '0%',
				barCategoryGap: '0%',
				itemStyle: { borderRadius: [r, r, 0, 0], color: gradientOf(color) },
				emphasis: { focus: 'series' },
				blur: { itemStyle: { opacity: BAR_MOCKUP_TUNING.blurOpacity } },
				animationDelay: (idx: number): number =>
					idx * BAR_MOCKUP_TUNING.staggerMs +
					index * BAR_MOCKUP_TUNING.seriesStaggerMs,
				// ECharts bar는 x값 중심 정렬 → 버킷 중심에 놓아야 [edge, edge+size)를 채운다.
				// 선행 유령 널빈은 count가 null이라 렌더되지 않는다(R3)
				data: edges.map(
					(edge, i) =>
						[edge + half, counts[i] ?? null] as [number, number | null],
				),
			};
		})
		.filter((s): s is HistogramSeriesItem => s !== null);

	if (series.length > 0 && markLine) {
		series[0].markLine = markLine;
	}

	const yAxisUnit = widget.yAxisUnit ?? 'none';

	const option = {
		animation: !reducedMotion,
		animationDuration: BAR_MOCKUP_TUNING.entryDurationMs,
		animationDurationUpdate: BAR_MOCKUP_TUNING.updateDurationMs,
		animationEasing: 'cubicOut',
		grid: { left: 8, right: 8, top: 12, bottom: 8, containLabel: true },
		// 범례는 React Legend가 담당하고 LegendComponent는 미등록(echartsCore)이라
		// echarts legend 옵션은 넣지 않는다(2a 라인·막대 빌더와 동일 — 죽은 설정).
		xAxis: {
			// 값 x축 — 버킷 위치. 시간·timezone 무관
			type: 'value',
			scale: true,
			// M4: extent를 버킷 경계로 고정한다. 기본(데이터 extent)은 버킷 "중심"
			// 기준이라 첫/마지막 막대의 바깥 절반이 grid 밖으로 잘릴 수 있다.
			min: edges.length > 0 ? edges[0] : undefined,
			max:
				edges.length > 0 ? edges[edges.length - 1] + bucketSize : undefined,
			axisLabel: { hideOverlap: true },
		},
		yAxis: {
			// 카운트 축: 막대와 동일하게 0 기준선 고정(높이 왜곡 방지)
			type: 'value',
			scale: false,
			min: 0,
			axisLabel: {
				formatter: (value: number): string =>
					getYAxisFormattedValue(String(value), yAxisUnit, widget.decimalPrecision),
			},
		},
		axisPointer: {
			// 2a와 동일한 선형 포인터 기본값.
			// snap:true라 updateAxisPointer 페이로드의 x값은 raw 좌표가 아니라
			// 가장 가까운 데이터 x(=버킷 중심)로 스냅되어 온다(axisTrigger.js:155-160).
			// resolveBucketIndex는 중심값도 같은 버킷으로 해석하므로 안전하다(M3).
			show: true,
			snap: true,
			lineStyle: { color: isDarkMode ? '#5f6570' : '#9aa0a6' },
		},
		series,
	};

	// echarts SeriesOption 유니온이 커서 인라인 리터럴을 바로 EChartsOption으로
	// 선언하면 초과 프로퍼티 검사에 걸린다. 경계에서 한 번만 캐스팅한다.
	return { option: option as EChartsOption, seriesLabels };
}
