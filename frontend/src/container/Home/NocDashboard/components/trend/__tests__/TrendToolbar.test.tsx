import { fireEvent, render, screen } from '@testing-library/react';

import TrendToolbar from '../TrendToolbar';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

describe('TrendToolbar', () => {
	it('renders help ⓘ button with aria-label', () => {
		render(
			<TrendToolbar
				metric="err"
				onMetricChange={jest.fn()}
				logScale={false}
				onLogScaleChange={jest.fn()}
			/>,
		);
		expect(screen.getByLabelText('noc_c2_help_aria')).toBeInTheDocument();
	});

	it('shows log toggle on p99 and fires onLogScaleChange', () => {
		const onLog = jest.fn();
		render(
			<TrendToolbar
				metric="p99"
				onMetricChange={jest.fn()}
				logScale={false}
				onLogScaleChange={onLog}
			/>,
		);
		fireEvent.click(screen.getByText('noc_c2_log_scale'));
		expect(onLog).toHaveBeenCalledWith(true);
	});

	it('hides log toggle on err tab', () => {
		render(
			<TrendToolbar
				metric="err"
				onMetricChange={jest.fn()}
				logScale={false}
				onLogScaleChange={jest.fn()}
			/>,
		);
		expect(screen.queryByText('noc_c2_log_scale')).toBeNull();
	});
});
