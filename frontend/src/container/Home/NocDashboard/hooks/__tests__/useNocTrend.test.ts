import { TrendTarget } from '../../types';
import { parseTrendSeries } from '../useNocTrend';

const targets: TrendTarget[] = [
	{ name: 'cart', color: '#3987e5' },
	{ name: 'recommendation', color: '#199e70' },
];

// 정규화 v3 result shape 모사 (Step 1에서 확정한 실제 경로로 맞출 것).
// A=count(rps), B=count+has_error(err), C=p99
function result(queryName: string, service: string, vals: [number, string][]) {
	return { queryName, metric: { 'service.name': service }, values: vals };
}

describe('parseTrendSeries', () => {
	it('err metric: B/A*100, B-missing service joins as 0', () => {
		const payload = {
			result: [
				result('A', 'cart', [[100, '10'], [160, '20']]),
				result('A', 'recommendation', [[100, '5'], [160, '5']]),
				// B: only recommendation has errors; cart has NO B series (오류 0)
				result('B', 'recommendation', [[100, '5'], [160, '5']]),
				result('C', 'cart', [[100, '1000000'], [160, '2000000']]),
				result('C', 'recommendation', [[100, '3000000'], [160, '3000000']]),
			],
		};
		const series = parseTrendSeries(payload, targets, 'err', 60);
		const cart = series.find((s) => s.name === 'cart')!;
		const rec = series.find((s) => s.name === 'recommendation')!;
		expect(cart.points.map((p) => p.v)).toEqual([0, 0]); // B 결측 -> 0
		expect(rec.points.map((p) => p.v)).toEqual([100, 100]); // 5/5*100
		expect(cart.color).toBe('#3987e5'); // 엔티티 색 고정
	});

	it('p99 metric: ns -> ms via /1e6', () => {
		const payload = {
			result: [result('C', 'cart', [[100, '599964431525']])],
		};
		const series = parseTrendSeries(payload, [targets[0]], 'p99', 60);
		expect(series[0].points[0].v).toBeCloseTo(599964.43, 1);
	});

	it('rps metric: A / stepSec', () => {
		const payload = { result: [result('A', 'cart', [[100, '120']])] };
		const series = parseTrendSeries(payload, [targets[0]], 'rps', 60);
		expect(series[0].points[0].v).toBeCloseTo(2, 5); // 120/60
	});

	it('target with no series at all is marked missing (선 생략, 범례 보존)', () => {
		const payload = { result: [result('A', 'cart', [[100, '10']])] };
		const series = parseTrendSeries(payload, targets, 'rps', 60);
		const rec = series.find((s) => s.name === 'recommendation')!;
		expect(rec.missing).toBe(true);
		expect(rec.points).toEqual([]);
	});
});
