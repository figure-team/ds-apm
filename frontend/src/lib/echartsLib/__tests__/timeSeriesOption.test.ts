import { getYAxisFormattedValue } from 'components/Graph/yAxisConfig';
import {
	getTimezoneObjectByTimezoneString,
	UTC_TIMEZONE,
} from 'components/CustomTimePicker/timezoneUtils';
import { LineInterpolation } from 'lib/uPlotV2/config/types';

import { buildTimeSeriesOption } from '../builders/timeSeriesOption';

const apiResponse = {
	data: {
		result: [
			{
				metric: { __name__: 'latency' },
				queryName: 'A',
				legend: '',
				values: [[1700000000, '100'], [1700000030, '200']],
			},
			{
				metric: { __name__: 'errors' },
				queryName: 'B',
				legend: '',
				values: [[1700000000, '1'], [1700000030, '2']],
			},
		],
		resultType: 'matrix',
	},
} as never;

const chartData = [
	[1700000000, 1700000030],
	[100, 200],
	[1, 2],
] as never;

// as never로 감싸기 전 원본을 남겨 spread(...baseWidget)에 재사용한다.
// (never를 바로 spread하면 TS2698 - spread는 object 타입에서만 가능)
const baseWidget = {
	id: 'w1',
	thresholds: [
		{ thresholdValue: 500, thresholdUnit: 'ms', thresholdLabel: '경고', thresholdColor: 'orange' },
	],
	yAxisUnit: 'ms',
	customLegendColors: {},
};

const baseArgs = {
	widget: baseWidget as never,
	apiResponse,
	chartData,
	currentQuery: undefined as never,
	isDarkMode: true,
	reducedMotion: false,
	minTimeScale: 1700000000,
	maxTimeScale: 1700000030,
};

describe('buildTimeSeriesOption', () => {
	it('시리즈 id는 인덱스+라벨로 유일·안정, name은 라벨 (morphing·중복 라벨 대비)', () => {
		const { option, seriesLabels } = buildTimeSeriesOption(baseArgs);
		const series = (option as { series: Array<{ id: string; name: string }> }).series;
		expect(series).toHaveLength(2);
		expect(series[0].id).toBe(`0:${seriesLabels[0]}`);
		expect(series[1].id).toBe(`1:${seriesLabels[1]}`);
		expect(series[0].name).toBe(seriesLabels[0]);
		expect(seriesLabels[0]).not.toBe(seriesLabels[1]);
	});

	it('타임스탬프를 ms로 변환한다 (uPlot은 초, echarts time축은 ms)', () => {
		const { option } = buildTimeSeriesOption(baseArgs);
		const first = (option as { series: Array<{ data: [number, number][] }> })
			.series[0].data[0];
		expect(first[0]).toBe(1700000000 * 1000);
	});

	it('임계값이 markLine으로 변환된다 (단위 변환 포함)', () => {
		const { option } = buildTimeSeriesOption(baseArgs);
		const markLine = (option as {
			series: Array<{ markLine?: { data: Array<{ yAxis: number }> } }>;
		}).series[0].markLine;
		expect(markLine).toBeDefined();
		// ms → ms 동일 단위이므로 500 그대로
		expect(markLine?.data[0].yAxis).toBe(500);
	});

	it('reducedMotion이면 애니메이션이 꺼진다', () => {
		const { option } = buildTimeSeriesOption({ ...baseArgs, reducedMotion: true });
		expect((option as { animation: boolean }).animation).toBe(false);
	});

	it('x축 범위가 min/maxTimeScale(ms)로 고정된다', () => {
		const { option } = buildTimeSeriesOption(baseArgs);
		const xAxis = (option as { xAxis: { min: number; max: number } }).xAxis;
		expect(xAxis.min).toBe(1700000000000);
		expect(xAxis.max).toBe(1700000030000);
	});

	// 추가 테스트 6종 (리뷰 반영)

	it('① isLogScale이면 yAxis.type이 log', () => {
		const { option } = buildTimeSeriesOption({
			...baseArgs,
			widget: { ...baseWidget, isLogScale: true } as never,
		});
		const yAxis = (option as { yAxis: { type: string } }).yAxis;
		expect(yAxis.type).toBe('log');
	});

	it('② visibilityMap으로 숨긴 시리즈는 option.series에서 제외되고 markLine은 남은 첫 표시 시리즈로 이동', () => {
		const { option, seriesLabels } = buildTimeSeriesOption({
			...baseArgs,
			visibilityMap: { 1: false },
		});
		const series = (option as {
			series: Array<{ id: string; name: string; markLine?: unknown }>;
		}).series;
		expect(series).toHaveLength(1);
		expect(series[0].name).toBe(seriesLabels[1]);
		expect(series[0].markLine).toBeDefined();
	});

	it('③ 유효 포인트가 1개뿐인 시리즈는 showSymbol이 강제로 true (라인 미표시로 사라짐 방지)', () => {
		const singlePointChartData = [
			[1700000000, 1700000030],
			[100, null],
			[1, 2],
		] as never;
		const { option } = buildTimeSeriesOption({
			...baseArgs,
			chartData: singlePointChartData,
		});
		const series = (option as {
			series: Array<{ showSymbol: boolean }>;
		}).series;
		expect(series[0].showSymbol).toBe(true);
		expect(series[1].showSymbol).toBe(false);
	});

	it('④ lineInterpolation: StepAfter → step은 end, smooth는 false', () => {
		const { option } = buildTimeSeriesOption({
			...baseArgs,
			widget: {
				...baseWidget,
				lineInterpolation: LineInterpolation.StepAfter,
			} as never,
		});
		const series = (option as {
			series: Array<{ step?: string; smooth: boolean }>;
		}).series;
		expect(series[0].step).toBe('end');
		expect(series[0].smooth).toBe(false);
	});

	it('⑤ yAxis.axisLabel.formatter가 getYAxisFormattedValue와 동일한 결과를 낸다', () => {
		const { option } = buildTimeSeriesOption(baseArgs);
		const yAxis = (option as {
			yAxis: { axisLabel: { formatter: (v: number) => string } };
		}).yAxis;
		expect(yAxis.axisLabel.formatter(1500)).toBe(
			getYAxisFormattedValue('1500', 'ms', undefined),
		);
	});

	it('⑥ timezone 지정 시 x축 formatter가 해당 시간대로 포맷한다 (UTC vs Asia/Seoul 상이)', () => {
		const asiaSeoul = getTimezoneObjectByTimezoneString('Asia/Seoul');
		const { option: utcOption } = buildTimeSeriesOption({
			...baseArgs,
			timezone: UTC_TIMEZONE,
		});
		const { option: seoulOption } = buildTimeSeriesOption({
			...baseArgs,
			timezone: asiaSeoul,
		});
		const utcXAxis = (utcOption as {
			xAxis: { axisLabel: { formatter: (v: number) => string } };
		}).xAxis;
		const seoulXAxis = (seoulOption as {
			xAxis: { axisLabel: { formatter: (v: number) => string } };
		}).xAxis;
		const sampleMs = 1700000000000;
		expect(utcXAxis.axisLabel.formatter(sampleMs)).not.toBe(
			seoulXAxis.axisLabel.formatter(sampleMs),
		);
	});

	// spanGaps 매핑 (echarts connectNulls는 all-or-nothing)

	it('⑦ spanGaps 미지정(기본 true)이면 connectNulls는 true', () => {
		const { option } = buildTimeSeriesOption(baseArgs);
		const series = (option as { series: Array<{ connectNulls: boolean }> })
			.series;
		expect(series[0].connectNulls).toBe(true);
	});

	it('⑧ spanGaps=false면 connectNulls는 false (모든 갭에서 끊김)', () => {
		const { option } = buildTimeSeriesOption({
			...baseArgs,
			widget: { ...baseWidget, spanGaps: false } as never,
		});
		const series = (option as { series: Array<{ connectNulls: boolean }> })
			.series;
		expect(series[0].connectNulls).toBe(false);
	});

	it('⑨ spanGaps 숫자면 임계 초과 시간 갭에 null 브레이크가 삽입되고 connectNulls는 false', () => {
		// 갭 30 > 임계 10 → 중간 지점에 null 삽입 → 포인트 2개→3개
		const { option } = buildTimeSeriesOption({
			...baseArgs,
			widget: { ...baseWidget, spanGaps: 10 } as never,
		});
		const series = (option as {
			series: Array<{ connectNulls: boolean; data: Array<[number, number | null]> }>;
		}).series;
		expect(series[0].connectNulls).toBe(false);
		expect(series[0].data).toHaveLength(3);
		expect(series[0].data.some(([, v]) => v === null)).toBe(true);
	});
});
