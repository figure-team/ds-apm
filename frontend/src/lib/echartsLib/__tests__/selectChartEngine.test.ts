import {
	ENGINE_LIMITS,
	evaluateChartEngine,
} from '../selectChartEngine';

// [timestamps, ...series] 형태의 AlignedData 목 생성
function makeData(seriesCount: number, pointCount: number): number[][] {
	const ts = Array.from({ length: pointCount }, (_, i) => i);
	return [ts, ...Array.from({ length: seriesCount }, () => ts.map(() => 1))];
}

describe('evaluateChartEngine', () => {
	it('경계 이하는 echarts', () => {
		expect(
			evaluateChartEngine({ data: makeData(30, 100) as never }),
		).toBe('echarts');
	});

	it('시리즈 수 초과는 uplot 강등', () => {
		expect(
			evaluateChartEngine({ data: makeData(31, 100) as never }),
		).toBe('uplot');
	});

	it('총 포인트 초과는 uplot 강등 (10시리즈 × 5001포인트 > 50000)', () => {
		expect(
			evaluateChartEngine({ data: makeData(10, 5001) as never }),
		).toBe('uplot');
	});

	it('총 포인트가 정확히 50000(경계)이면 echarts 유지 (10시리즈 × 5000포인트)', () => {
		expect(
			evaluateChartEngine({ data: makeData(10, 5000) as never }),
		).toBe('echarts');
	});

	it('override는 판정보다 우선한다', () => {
		expect(
			evaluateChartEngine({ data: makeData(100, 1000) as never, override: 'echarts' }),
		).toBe('echarts');
	});

	it('히스테리시스: 강등 후 임계 80~100% 구간에서는 uplot 유지', () => {
		// 25시리즈 = maxSeries(30)의 83% → 복귀 조건(80% 미만) 불충족
		expect(
			evaluateChartEngine({ data: makeData(25, 100) as never, previous: 'uplot' }),
		).toBe('uplot');
	});

	it('히스테리시스: 임계 80% 미만으로 내려오면 echarts 복귀', () => {
		// 23시리즈 < 30*0.8=24, 총 포인트 2300 < 40000
		expect(
			evaluateChartEngine({ data: makeData(23, 100) as never, previous: 'uplot' }),
		).toBe('echarts');
	});

	it('빈 데이터는 echarts (렌더 전 게이트는 훅이 담당)', () => {
		expect(evaluateChartEngine({ data: [[]] as never })).toBe('echarts');
	});

	it('상수 스냅샷 (임계치 변경 시 의도 확인용)', () => {
		expect(ENGINE_LIMITS).toEqual({
			maxSeries: 30,
			maxTotalPoints: 50000,
			recoveryRatio: 0.8,
		});
	});
});
