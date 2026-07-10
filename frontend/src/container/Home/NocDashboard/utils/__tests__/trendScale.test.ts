import { TrendSeries } from '../../types';
import { computeScale, formatValue, makeMapper, Pad, TICKS } from '../trendScale';

const PAD0: Pad = { top: 0, right: 0, bottom: 0, left: 0 };

const series: TrendSeries[] = [
	{
		name: 'cart',
		color: '#000',
		points: [
			{ t: 0, v: 1 },
			{ t: 100, v: 1000 },
		],
	},
];

describe('computeScale', () => {
	it('exposes minPos as the smallest positive value', () => {
		const s = computeScale(series, 'p99');
		expect(s.minPos).toBe(1);
	});

	it('minPos is undefined when no positive samples', () => {
		const s = computeScale(
			[{ name: 'z', color: '#000', points: [{ t: 0, v: 0 }] }],
			'rps',
		);
		expect(s.minPos).toBeUndefined();
	});
});

describe('makeMapper linear', () => {
	it('maps minV to bottom and maxV to top', () => {
		const scale = { minT: 0, maxT: 100, minV: 0, maxV: 1100, minPos: 1 };
		const m = makeMapper(scale, 100, 100, PAD0, 'p99', false);
		expect(m.y(0)).toBeCloseTo(100, 5);
		expect(m.y(1100)).toBeCloseTo(0, 5);
		expect(m.yTicks()).toHaveLength(TICKS + 1);
	});
});

describe('makeMapper log', () => {
	const scale = { minT: 0, maxT: 100, minV: 0, maxV: 1000, minPos: 1 };

	it('maps minPos to bottom, maxV to top, geometric mean to middle', () => {
		const m = makeMapper(scale, 100, 100, PAD0, 'p99', true);
		expect(m.y(1)).toBeCloseTo(100, 5);
		expect(m.y(1000)).toBeCloseTo(0, 5);
		expect(m.y(Math.sqrt(1000))).toBeCloseTo(50, 3); // log 중간점
	});

	it('clamps zero/negative values to the bottom', () => {
		const m = makeMapper(scale, 100, 100, PAD0, 'p99', true);
		expect(m.y(0)).toBeCloseTo(100, 5);
	});

	it('err metric ignores logScale (linear kept)', () => {
		const lin = makeMapper(scale, 100, 100, PAD0, 'err', false);
		const log = makeMapper(scale, 100, 100, PAD0, 'err', true);
		expect(log.y(500)).toBeCloseTo(lin.y(500), 5);
	});

	it('falls back to linear when minPos is undefined', () => {
		const noPos = { minT: 0, maxT: 100, minV: 0, maxV: 1000 };
		const m = makeMapper(noPos, 100, 100, PAD0, 'p99', true);
		expect(m.y(0)).toBeCloseTo(100, 5); // 선형: minV=0이 바닥
		expect(m.y(500)).toBeCloseTo(50, 5);
	});
});

describe('formatValue', () => {
	it('err: 소수 1자리 퍼센트', () => {
		expect(formatValue(3.14159, 'err')).toBe('3.1');
	});

	it('1 미만 값은 소수 2자리 — 저트래픽 RPS 로그 눈금이 "0"으로 뭉개지지 않음', () => {
		expect(formatValue(0.016, 'rps')).toBe('0.02');
	});

	it('10 미만 값은 소수 1자리(정수는 그대로)', () => {
		expect(formatValue(8.34, 'p99')).toBe('8.3');
		expect(formatValue(8, 'rps')).toBe('8');
	});

	it('10 이상은 정수 반올림', () => {
		expect(formatValue(20844.4, 'p99')).toBe('20844');
	});

	it('0은 "0" 유지', () => {
		expect(formatValue(0, 'rps')).toBe('0');
	});
});
