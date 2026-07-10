import { fireEvent, render, screen } from '@testing-library/react';

import TrendLegend from '../TrendLegend';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

const items = [
	{ name: 'cart', color: '#3987e5', hidden: false },
	{ name: 'load-generator', color: '#199e70', hidden: true },
];

describe('TrendLegend', () => {
	it('click calls onToggle with the service name', () => {
		const onToggle = jest.fn();
		render(
			<TrendLegend
				items={items}
				hovered={null}
				onHover={jest.fn()}
				onToggle={onToggle}
			/>,
		);
		fireEvent.click(screen.getByText('cart'));
		expect(onToggle).toHaveBeenCalledWith('cart');
	});

	it('hidden item gets hidden class and aria-pressed=true', () => {
		render(
			<TrendLegend
				items={items}
				hovered={null}
				onHover={jest.fn()}
				onToggle={jest.fn()}
			/>,
		);
		const btn = screen.getByText('load-generator').closest('button')!;
		expect(btn.className).toContain('hidden');
		expect(btn).toHaveAttribute('aria-pressed', 'true');
		const visibleBtn = screen.getByText('cart').closest('button')!;
		expect(visibleBtn).toHaveAttribute('aria-pressed', 'false');
	});
});
