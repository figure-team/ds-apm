// 위젯 메타를 막대 ECharts option으로 변환. 시리즈 순서·라벨·markLine은
// 라인 빌더(timeSeriesOption)와 공유 규칙을 따른다. 막대 특이점: 0 기준선,
// 상하 그라데이션 채움, 상단 라운드(스택 시 최상단 표시 시리즈만), 스택.
import { Timezone } from 'components/CustomTimePicker/timezoneUtils';
import { getYAxisFormattedValue } from 'components/Graph/yAxisConfig';
import dayjs from 'dayjs';
import timezonePlugin from 'dayjs/plugin/timezone';
import utcPlugin from 'dayjs/plugin/utc';
import type uPlot from 'uplot';
import { Widgets } from 'types/api/dashboard/getAll';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import { EChartsOption } from '../echartsCore';
import { BAR_MOCKUP_TUNING, getSeriesColor } from '../themes/dsapmTheme';
import {
	buildMarkLine,
	resolveSeriesLabels,
	SEC_TO_MS,
} from './timeSeriesOption';

dayjs.extend(utcPlugin);
dayjs.extend(timezonePlugin);

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

interface BarSeriesItem {
	id: string;
	name: string;
	type: 'bar';
	stack?: string;
	barMaxWidth: number;
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

export function buildBarOption({
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
	const isStacked = Boolean(widget.stackedBarChart);
	const r = BAR_MOCKUP_TUNING.borderRadius;

	// 표시 시리즈의 원본 인덱스 목록 — 스택 최상단 라운드 판정용 (숨김 필터 반영)
	const visibleOriginalIdx = seriesLabels
		.map((_, i) => i)
		.filter((i) => !(visibilityMap && visibilityMap[i + 1] === false));
	const topVisibleIdx = isStacked
		? visibleOriginalIdx[visibleOriginalIdx.length - 1]
		: undefined;

	const series: BarSeriesItem[] = seriesLabels
		.map((label, index): BarSeriesItem | null => {
			const seriesIndex = index + 1; // uPlot 규약
			if (visibilityMap && visibilityMap[seriesIndex] === false) {
				return null;
			}
			const color = getSeriesColor(
				label,
				widget.customLegendColors ?? {},
				isDarkMode,
			);
			const values = (chartData[seriesIndex] ?? []) as (number | null)[];
			// 상단 라운드: 비스택=모든 시리즈, 스택=최상단 표시 시리즈만
			const roundTop = !isStacked || index === topVisibleIdx;
			const borderRadius = roundTop ? [r, r, 0, 0] : [0, 0, 0, 0];
			return {
				id: `${index}:${label}`,
				name: label,
				type: 'bar',
				stack: isStacked ? 'total' : undefined,
				barMaxWidth: BAR_MOCKUP_TUNING.barMaxWidth,
				itemStyle: {
					borderRadius,
					color: {
						type: 'linear',
						x: 0,
						y: 0,
						x2: 0,
						y2: 1,
						colorStops: [
							{ offset: 0, color: `${color}${BAR_MOCKUP_TUNING.areaAlphaTop}` },
							{ offset: 1, color: `${color}${BAR_MOCKUP_TUNING.areaAlphaBottom}` },
						],
					},
				},
				emphasis: { focus: 'series' },
				blur: { itemStyle: { opacity: BAR_MOCKUP_TUNING.blurOpacity } },
				animationDelay: (idx: number): number =>
					idx * BAR_MOCKUP_TUNING.staggerMs +
					index * BAR_MOCKUP_TUNING.seriesStaggerMs,
				data: timestamps.map(
					(ts, i) => [ts * SEC_TO_MS, values[i] ?? null] as [number, number | null],
				),
			};
		})
		.filter((s): s is BarSeriesItem => s !== null);

	// 임계선은 첫 표시 시리즈에만 (숨김 필터 이후 기준)
	if (series.length > 0 && markLine) {
		series[0].markLine = markLine;
	}

	const yAxisUnit = widget.yAxisUnit ?? 'none';
	const softMax = widget.softMax ?? undefined;
	const isLog = Boolean(widget.isLogScale);

	const option = {
		animation: !reducedMotion,
		animationDuration: BAR_MOCKUP_TUNING.entryDurationMs,
		animationDurationUpdate: BAR_MOCKUP_TUNING.updateDurationMs,
		animationEasing: 'cubicOut',
		animationDelayUpdate: (idx: number): number => idx * 2,
		grid: { left: 8, right: 8, top: 12, bottom: 8, containLabel: true },
		// 범례는 React Legend가 담당하고 LegendComponent는 미등록(echartsCore)이라
		// echarts legend 옵션은 넣지 않는다(라인 빌더와 동일 — 넣어도 무시되는 죽은 설정).
		xAxis: {
			type: 'time',
			min: minTimeScale !== undefined ? minTimeScale * SEC_TO_MS : undefined,
			max: maxTimeScale !== undefined ? maxTimeScale * SEC_TO_MS : undefined,
			axisLabel: {
				formatter: (value: number): string =>
					timezone
						? dayjs(value).tz(timezone.value).format('HH:mm:ss')
						: dayjs(value).format('HH:mm:ss'),
				hideOverlap: true,
			},
		},
		yAxis: {
			// 막대 0-기준선(스펙 §3.3): 비로그축은 scale:false + min:0(왜곡 방지).
			// softMin은 막대 기준선(0)과 충돌하므로 미적용. softMax는 상단 확장만 유지.
			// 주의: min:0은 음수 막대값을 클리핑한다(막대는 양수 전제). 로그축은 0/음수
			// 미지원(라인 빌더와 동일 caveat) — Task 6 실측에서 0 포함 데이터 동작 확인.
			type: isLog ? 'log' : 'value',
			scale: false,
			min: isLog ? undefined : 0,
			max:
				softMax !== undefined
					? (extent: { min: number; max: number }): number =>
							Math.max(extent.max, softMax)
					: undefined,
			axisLabel: {
				formatter: (value: number): string =>
					getYAxisFormattedValue(String(value), yAxisUnit, widget.decimalPrecision),
			},
		},
		axisPointer: {
			// 스펙 §4.2: 선형 포인터 기본(1단계 패리티). shadow는 시각검증 후 별도 결정.
			show: true,
			snap: true,
			lineStyle: { color: isDarkMode ? '#5f6570' : '#9aa0a6' },
		},
		series,
	};

	// echarts의 실제 SeriesOption 등은 등록된 컴포넌트 조합에 따라 매우 큰
	// 유니온 타입이라 인라인 리터럴을 바로 EChartsOption으로 선언하면 불필요한
	// 초과 프로퍼티 검사에 걸린다. 여기서 한 번만 캐스팅해 계약(반환 타입)은
	// 유지하면서 내부 구현은 구조적으로 느슨하게 둔다.
	return { option: option as EChartsOption, seriesLabels };
}
