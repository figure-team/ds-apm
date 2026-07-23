import { buildUPlotShim } from '../utils/uplotShim';

describe('buildUPlotShim', () => {
	const chartData = [
		[1700000000, 1700000030],
		[100, 200],
	] as never;

	it('data와 series 표면을 제공한다', () => {
		const shim = buildUPlotShim(
			chartData,
			[{ label: 'p99', color: '#D81B2C', show: true }],
			0,
		);
		expect(shim.data).toBe(chartData);
		// uPlot 규약: series[0]은 x축 자리
		expect(shim.series).toHaveLength(2);
		expect(shim.series[1].label).toBe('p99');
		expect(shim.series[1].show).toBe(true);
	});

	it('cursor.idx에 hover dataIndex를 반영한다 (타임스탬프 헤더 출처 — 리뷰 반영)', () => {
		const shim = buildUPlotShim(
			chartData,
			[{ label: 'p99', color: '#D81B2C', show: true }],
			1,
		);
		expect(shim.cursor.idx).toBe(1);
	});

	it('숨김 시리즈는 show:false로 전달된다 (툴팁 목록에서 제외됨)', () => {
		const shim = buildUPlotShim(
			chartData,
			[{ label: 'p99', color: '#D81B2C', show: false }],
			0,
		);
		expect(shim.series[1].show).toBe(false);
	});

	it('stroke는 함수형 호출을 지원한다 (uPlot series.stroke 규약)', () => {
		const shim = buildUPlotShim(
			chartData,
			[{ label: 'p99', color: '#D81B2C', show: true }],
			0,
		);
		const stroke = shim.series[1].stroke as () => string;
		expect(stroke()).toBe('#D81B2C');
	});
});
