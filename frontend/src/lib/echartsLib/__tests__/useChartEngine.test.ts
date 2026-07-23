import { renderHook } from '@testing-library/react';

import { useChartEngine } from '../hooks/useChartEngine';

// [timestamps, ...series] 형태의 AlignedData 목 생성 (selectChartEngine.test.ts와 동일 헬퍼)
function makeData(seriesCount: number, pointCount: number): number[][] {
	const ts = Array.from({ length: pointCount }, (_, i) => i);
	return [ts, ...Array.from({ length: seriesCount }, () => ts.map(() => 1))];
}

describe('useChartEngine', () => {
	it('데이터가 undefined/빈 배열이면 null을 반환한다', () => {
		const { result: undefinedResult } = renderHook(() =>
			useChartEngine(undefined),
		);
		expect(undefinedResult.current).toBeNull();

		const { result: emptyResult } = renderHook(() =>
			useChartEngine([[]] as never),
		);
		expect(emptyResult.current).toBeNull();
	});

	it('데이터 도착 후 경계 이하면 echarts로 판정한다', () => {
		const { result } = renderHook(() =>
			useChartEngine(makeData(10, 100) as never),
		);
		expect(result.current).toBe('echarts');
	});

	it('forceFallback이 1회 true였다가 false로 돌아오고 데이터가 소규모여도 uplot을 유지한다 (래치)', () => {
		const { result, rerender } = renderHook(
			({ forceFallback }: { forceFallback?: boolean }) =>
				useChartEngine(makeData(5, 50) as never, undefined, forceFallback),
			{ initialProps: { forceFallback: true } },
		);
		expect(result.current).toBe('uplot');

		// forceFallback을 false로 되돌리고, echarts 복귀 조건을 만족하는 소규모 데이터로 재렌더
		rerender({ forceFallback: false });
		expect(result.current).toBe('uplot');
	});

	it('대규모 데이터로 강등된 후 임계 80~100% 구간 데이터로 재렌더해도 uplot을 유지한다 (렌더 간 히스테리시스)', () => {
		const { result, rerender } = renderHook(
			({ data }: { data: number[][] }) => useChartEngine(data as never),
			{ initialProps: { data: makeData(31, 100) } },
		);
		expect(result.current).toBe('uplot');

		// 25시리즈 = maxSeries(30)의 83% → 복귀 조건(80% 미만) 불충족, uplot 유지가 기대값
		rerender({ data: makeData(25, 100) });
		expect(result.current).toBe('uplot');
	});
});
