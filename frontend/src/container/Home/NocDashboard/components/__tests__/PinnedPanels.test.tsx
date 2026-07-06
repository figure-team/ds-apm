import { fireEvent, render, screen } from '@testing-library/react';

import { NocPinnedSlot } from '../../types';
import PinnedPanels from '../PinnedPanels';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('container/GridCardLayout/GridCard', () => ({
	__esModule: true,
	default: ({ widget }: { widget: { id: string } }): JSX.Element => (
		<div data-testid={`grid-card-${widget.id}`} />
	),
}));

const SLOT: NocPinnedSlot = {
	ref: { dashboardId: 'd1', widgetId: 'w1' },
	dashboardTitle: '결제 구간 모니터링',
	version: 'v5',
	widget: { id: 'w1', panelTypes: 'graph', title: '결제 지연 p95' } as never,
};

describe('PinnedPanels', () => {
	it('renders GridCard per pinned slot with source caption', () => {
		render(
			<PinnedPanels slots={[SLOT]} onUnpin={jest.fn()} onOpenPicker={jest.fn()} />,
		);
		expect(screen.getByTestId('grid-card-w1')).toBeInTheDocument();
		expect(screen.getByText('결제 구간 모니터링')).toBeInTheDocument();
	});

	it('renders missing placeholder when widget is null', () => {
		render(
			<PinnedPanels
				slots={[{ ...SLOT, widget: null }]}
				onUnpin={jest.fn()}
				onOpenPicker={jest.fn()}
			/>,
		);
		expect(screen.getByText('noc_c2_pin_missing')).toBeInTheDocument();
	});

	it('unpin button calls onUnpin, add tile calls onOpenPicker', () => {
		const onUnpin = jest.fn();
		const onOpenPicker = jest.fn();
		render(
			<PinnedPanels slots={[SLOT]} onUnpin={onUnpin} onOpenPicker={onOpenPicker} />,
		);
		fireEvent.click(screen.getByRole('button', { name: 'noc_c2_pin_unpin' }));
		expect(onUnpin).toHaveBeenCalledWith('w1');
		fireEvent.click(screen.getByRole('button', { name: /noc_c2_pin_add/ }));
		expect(onOpenPicker).toHaveBeenCalled();
	});

	it('hides add tile when slots are full', () => {
		render(
			<PinnedPanels
				slots={[SLOT, SLOT, SLOT, SLOT]}
				onUnpin={jest.fn()}
				onOpenPicker={jest.fn()}
			/>,
		);
		expect(screen.queryByRole('button', { name: /noc_c2_pin_add/ })).toBeNull();
	});
});
