import { fireEvent, render, screen } from '@testing-library/react';

import { ResolvedTrendSeries } from '../../../types';
import TrendPlot from '../TrendPlot';

// jsdom엔 ResizeObserver가 없어 기본 크기 860x300으로 렌더된다(컴포넌트 가드 의존).
const series: ResolvedTrendSeries[] = [
	{
		name: 'cart',
		color: '#3987e5',
		resolvedColor: '#3987e5',
		points: [
			{ t: 0, v: 5 },
			{ t: 60000, v: 15 },
		],
	},
	{
		name: 'auth',
		color: '#199e70',
		resolvedColor: '#199e70',
		points: [
			{ t: 0, v: 1 },
			{ t: 60000, v: 8 },
		],
	},
];

describe('TrendPlot crosshair tooltip', () => {
	it('mousemove snaps to nearest timestamp and lists per-service values (desc)', () => {
		const { container } = render(
			<TrendPlot series={series} metric="rps" logScale={false} hovered={null} />,
		);
		const body = container.querySelector('.noc-c2-trend-body') as HTMLElement;
		// 기본 폭 860, PAD.left=52, PAD.right=170 → 플롯 x∈[52,690]. 600은 t=60000에 근접.
		fireEvent.mouseMove(body, { clientX: 600, clientY: 100 });
		expect(container.querySelector('.noc-c2-crosshair')).not.toBeNull();
		const rows = container.querySelectorAll('.noc-c2-tip-row');
		expect(rows).toHaveLength(2);
		expect(rows[0].textContent).toContain('cart'); // 15 > 8 → cart 먼저
		expect(rows[0].textContent).toContain('15');
		expect(rows[1].textContent).toContain('auth');
	});

	it('mouseleave clears the crosshair', () => {
		const { container } = render(
			<TrendPlot series={series} metric="rps" logScale={false} hovered={null} />,
		);
		const body = container.querySelector('.noc-c2-trend-body') as HTMLElement;
		fireEvent.mouseMove(body, { clientX: 600, clientY: 100 });
		fireEvent.mouseLeave(body);
		expect(container.querySelector('.noc-c2-crosshair')).toBeNull();
		expect(container.querySelector('.noc-c2-trend-tip')).toBeNull();
	});

	it('mousemove outside plot area (gutter) hides the tooltip', () => {
		const { container } = render(
			<TrendPlot series={series} metric="rps" logScale={false} hovered={null} />,
		);
		const body = container.querySelector('.noc-c2-trend-body') as HTMLElement;
		fireEvent.mouseMove(body, { clientX: 800, clientY: 100 }); // 우측 라벨 거터
		expect(container.querySelector('.noc-c2-crosshair')).toBeNull();
	});
});
