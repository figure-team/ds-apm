import { buildBarOption } from '../builders/barOption';
import { BAR_MOCKUP_TUNING } from '../themes/dsapmTheme';

const apiResponse = {
	data: {
		result: [
			{ metric: { __name__: 'a' }, queryName: 'A', legend: '', values: [[1700000000, '10'], [1700000030, '20']] },
			{ metric: { __name__: 'b' }, queryName: 'B', legend: '', values: [[1700000000, '1'], [1700000030, '2']] },
		],
		resultType: 'matrix',
	},
} as never;

const chartData = [[1700000000, 1700000030], [10, 20], [1, 2]] as never;

const baseWidget = {
	id: 'w1',
	thresholds: [{ thresholdValue: 500, thresholdUnit: 'ms', thresholdLabel: '경고', thresholdColor: 'orange' }],
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

type BarSeries = {
	id: string;
	name: string;
	type: string;
	stack?: string;
	barMaxWidth: number;
	itemStyle: { borderRadius: number[]; color: { colorStops: Array<{ offset: number; color: string }> } };
	emphasis: { focus: string };
	blur: { itemStyle: { opacity: number } };
	data: Array<[number, number | null]>;
	markLine?: unknown;
};
const seriesOf = (o: unknown): BarSeries[] => (o as { series: BarSeries[] }).series;

describe('buildBarOption', () => {
	it('시리즈는 bar 타입, 안정 id(index:label), name=라벨', () => {
		const { option, seriesLabels } = buildBarOption(baseArgs);
		const series = seriesOf(option);
		expect(series).toHaveLength(2);
		expect(series[0].type).toBe('bar');
		expect(series[0].id).toBe(`0:${seriesLabels[0]}`);
		expect(series[1].id).toBe(`1:${seriesLabels[1]}`);
		expect(series[0].name).toBe(seriesLabels[0]);
	});

	it('타임스탬프를 ms로 변환한다', () => {
		const series = seriesOf(buildBarOption(baseArgs).option);
		expect(series[0].data[0][0]).toBe(1700000000 * 1000);
	});

	it('비스택: stack 미설정, 모든 시리즈 상단 라운드', () => {
		const series = seriesOf(buildBarOption(baseArgs).option);
		const r = BAR_MOCKUP_TUNING.borderRadius;
		expect(series[0].stack).toBeUndefined();
		expect(series[0].itemStyle.borderRadius).toEqual([r, r, 0, 0]);
		expect(series[1].itemStyle.borderRadius).toEqual([r, r, 0, 0]);
	});

	it('스택: stack=total, 최상단 표시 시리즈만 상단 라운드', () => {
		const widget = { ...baseWidget, stackedBarChart: true };
		const series = seriesOf(buildBarOption({ ...baseArgs, widget: widget as never }).option);
		const r = BAR_MOCKUP_TUNING.borderRadius;
		expect(series[0].stack).toBe('total');
		expect(series[0].itemStyle.borderRadius).toEqual([0, 0, 0, 0]); // 아래 세그먼트
		expect(series[1].itemStyle.borderRadius).toEqual([r, r, 0, 0]); // 최상단
	});

	it('스택 + 최상단 숨김: 라운드가 다음 표시 시리즈로 이동', () => {
		const widget = { ...baseWidget, stackedBarChart: true };
		const series = seriesOf(
			buildBarOption({ ...baseArgs, widget: widget as never, visibilityMap: { 2: false } }).option,
		);
		const r = BAR_MOCKUP_TUNING.borderRadius;
		// series[1](index1, seriesIndex2)은 숨김 제외 → series[0]만 남고 최상단
		expect(series).toHaveLength(1);
		expect(series[0].id).toBe(`0:${buildBarOption(baseArgs).seriesLabels[0]}`);
		expect(series[0].itemStyle.borderRadius).toEqual([r, r, 0, 0]);
	});

	it('스택 + 중간 시리즈 숨김: 최상단 라운드는 원래 최상단(index2)에 유지, 하단(index0)은 각짐', () => {
		const threeSeriesApiResponse = {
			data: {
				result: [
					{ metric: { __name__: 'a' }, queryName: 'A', legend: '', values: [[1700000000, '10'], [1700000030, '20']] },
					{ metric: { __name__: 'b' }, queryName: 'B', legend: '', values: [[1700000000, '1'], [1700000030, '2']] },
					{ metric: { __name__: 'c' }, queryName: 'C', legend: '', values: [[1700000000, '3'], [1700000030, '4']] },
				],
				resultType: 'matrix',
			},
		} as never;
		const threeSeriesChartData = [
			[1700000000, 1700000030],
			[10, 20],
			[1, 2],
			[3, 4],
		] as never;
		const widget = { ...baseWidget, stackedBarChart: true };
		const series = seriesOf(
			buildBarOption({
				...baseArgs,
				widget: widget as never,
				apiResponse: threeSeriesApiResponse,
				chartData: threeSeriesChartData,
				visibilityMap: { 2: false }, // 중간(index1, seriesIndex2) 숨김
			}).option,
		);
		const r = BAR_MOCKUP_TUNING.borderRadius;
		expect(series).toHaveLength(2);
		expect(series[0].id.startsWith('0:')).toBe(true);
		expect(series[1].id.startsWith('2:')).toBe(true);
		expect(series[0].itemStyle.borderRadius).toEqual([0, 0, 0, 0]); // 하단(index0)
		expect(series[1].itemStyle.borderRadius).toEqual([r, r, 0, 0]); // 원래 최상단(index2)
	});

	it('시안 A 상하 그라데이션 채움(colorStops alphaTop/Bottom)', () => {
		const series = seriesOf(buildBarOption(baseArgs).option);
		const stops = series[0].itemStyle.color.colorStops;
		expect(stops[0].offset).toBe(0);
		expect(stops[0].color.endsWith(BAR_MOCKUP_TUNING.areaAlphaTop)).toBe(true);
		expect(stops[1].color.endsWith(BAR_MOCKUP_TUNING.areaAlphaBottom)).toBe(true);
	});

	it('emphasis focus:series, blur opacity, barMaxWidth 적용', () => {
		const series = seriesOf(buildBarOption(baseArgs).option);
		expect(series[0].emphasis.focus).toBe('series');
		expect(series[0].blur.itemStyle.opacity).toBe(BAR_MOCKUP_TUNING.blurOpacity);
		expect(series[0].barMaxWidth).toBe(BAR_MOCKUP_TUNING.barMaxWidth);
	});

	it('markLine은 첫 표시 시리즈에만 부착(단위변환 적용)', () => {
		const series = seriesOf(buildBarOption(baseArgs).option);
		expect(series[0].markLine).toBeDefined();
		expect(series[1].markLine).toBeUndefined();
	});

	it('visibilityMap false 시리즈는 option에서 제외', () => {
		const series = seriesOf(buildBarOption({ ...baseArgs, visibilityMap: { 1: false } }).option);
		expect(series).toHaveLength(1);
		expect(series[0].id.startsWith('1:')).toBe(true);
	});

	it('비로그축 yAxis는 0 기준선(scale 없음, min 0)', () => {
		const option = buildBarOption(baseArgs).option as { yAxis: { type: string; scale?: boolean; min?: number } };
		expect(option.yAxis.type).toBe('value');
		expect(option.yAxis.scale).toBe(false);
		expect(option.yAxis.min).toBe(0);
	});

	it('로그축 위젯은 yAxis type=log', () => {
		const widget = { ...baseWidget, isLogScale: true };
		const option = buildBarOption({ ...baseArgs, widget: widget as never }).option as { yAxis: { type: string } };
		expect(option.yAxis.type).toBe('log');
	});
});
