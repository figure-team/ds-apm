import { buildHistogramOption } from '../builders/histogramOption';
import { BAR_MOCKUP_TUNING } from '../themes/dsapmTheme';

// 유령 널빈(-10) + 버킷 0,10,20. bucketSize=10
const chartData = [
	[-10, 0, 10, 20],
	[null, 5, 12, 4],
	[null, 2, 8, 3],
] as never;

const apiResponse = {
	data: {
		result: [
			{ metric: { __name__: 'a' }, queryName: 'A', legend: '', values: [] },
			{ metric: { __name__: 'b' }, queryName: 'B', legend: '', values: [] },
		],
		resultType: 'matrix',
	},
} as never;

const baseWidget = {
	id: 'w1',
	thresholds: [
		{ thresholdValue: 10, thresholdUnit: 'none', thresholdLabel: '경고', thresholdColor: 'orange' },
	],
	yAxisUnit: 'none',
	customLegendColors: {},
};

const baseArgs = {
	widget: baseWidget as never,
	apiResponse,
	chartData,
	currentQuery: undefined as never,
	isDarkMode: true,
	reducedMotion: false,
	isQueriesMerged: false,
};

type HSeries = {
	id: string;
	name: string;
	type: string;
	barWidth?: number | string;
	barGap?: string;
	barCategoryGap?: string;
	itemStyle: { borderRadius: number[]; color: unknown };
	data: Array<[number, number | null]>;
	markLine?: unknown;
};
const seriesOf = (o: unknown): HSeries[] => (o as { series: HSeries[] }).series;

describe('buildHistogramOption', () => {
	it('시리즈는 bar 타입, 안정 id, 값 x축 option', () => {
		const { option } = buildHistogramOption(baseArgs);
		const series = seriesOf(option);
		expect(series[0].type).toBe('bar');
		expect(series[0].id.startsWith('0:')).toBe(true);
		expect((option as { xAxis: { type: string } }).xAxis.type).toBe('value');
	});

	it('막대 x는 버킷 중심(edge + bucketSize/2)이다', () => {
		const series = seriesOf(buildHistogramOption(baseArgs).option);
		// edges[1]=0 → 중심 5
		expect(series[0].data[1][0]).toBe(5);
	});

	// R1(개정): barWidth는 픽셀 단위라 데이터 단위로 주면 안 된다. 두 gap을 0으로
	// 두면 ECharts가 bandWidth(=버킷 폭 픽셀)를 N등분한다. 실폭·정렬은 Task 6 실측.
	it('barWidth를 지정하지 않고 두 gap을 0%로 둔다 (R1)', () => {
		const series = seriesOf(buildHistogramOption(baseArgs).option);
		expect(series).toHaveLength(2);
		series.forEach((s) => {
			expect(s.barWidth).toBeUndefined();
			expect(s.barGap).toBe('0%');
			expect(s.barCategoryGap).toBe('0%');
		});
	});

	it('병합 모드: 단일 시리즈, 색은 브랜드 레드', () => {
		const mergedData = [[-10, 0, 10, 20], [null, 7, 9, 2]] as never;
		const { option } = buildHistogramOption({
			...baseArgs,
			chartData: mergedData,
			isQueriesMerged: true,
		});
		const series = seriesOf(option);
		expect(series).toHaveLength(1);
		expect(series[0].barWidth).toBeUndefined();
		expect(JSON.stringify(series[0].itemStyle.color)).toContain('#D81B2C');
	});

	it('visibilityMap으로 숨긴 시리즈는 option에서 빠진다 (폭 재계산은 ECharts 담당)', () => {
		const series = seriesOf(
			buildHistogramOption({ ...baseArgs, visibilityMap: { 2: false } }).option,
		);
		expect(series).toHaveLength(1);
		expect(series[0].id.startsWith('0:')).toBe(true);
	});

	it('x축 범위를 버킷 경계로 고정한다 — 양끝 막대 잘림 방지 (M4)', () => {
		const option = buildHistogramOption(baseArgs).option as {
			xAxis: { min: number; max: number };
		};
		expect(option.xAxis.min).toBe(-10); // edges[0]
		expect(option.xAxis.max).toBe(30); // lastEdge(20) + bucketSize(10)
	});

	it('y축은 카운트 0 기준선(scale false, min 0)', () => {
		const option = buildHistogramOption(baseArgs).option as {
			yAxis: { type: string; scale?: boolean; min?: number };
		};
		expect(option.yAxis.type).toBe('value');
		expect(option.yAxis.scale).toBe(false);
		expect(option.yAxis.min).toBe(0);
	});

	it('선행 널빈은 데이터에 null로 남아 렌더되지 않는다', () => {
		const series = seriesOf(buildHistogramOption(baseArgs).option);
		expect(series[0].data[0][1]).toBeNull();
	});

	it('시안 A 상단 라운드와 그라데이션을 적용한다', () => {
		const series = seriesOf(buildHistogramOption(baseArgs).option);
		const r = BAR_MOCKUP_TUNING.borderRadius;
		expect(series[0].itemStyle.borderRadius).toEqual([r, r, 0, 0]);
		expect(JSON.stringify(series[0].itemStyle.color)).toContain('colorStops');
	});

	it('markLine은 첫 표시 시리즈에만 부착', () => {
		const series = seriesOf(buildHistogramOption(baseArgs).option);
		expect(series[0].markLine).toBeDefined();
		expect(series[1].markLine).toBeUndefined();
	});
});
