import { fireEvent, render, screen } from '@testing-library/react';

import FiredAlertsBadge from '../FiredAlertsBadge';

const safeNavigate = jest.fn();
jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate }),
}));

describe('FiredAlertsBadge', () => {
	beforeEach(() => safeNavigate.mockClear());

	it('renders fired count and navigates to alerts on click', () => {
		render(<FiredAlertsBadge count={3} />);
		fireEvent.click(screen.getByRole('button', { name: /noc_c2_fired/ }));
		expect(safeNavigate).toHaveBeenCalledWith('/alerts');
	});

	it('uses quiet style when count is zero', () => {
		render(<FiredAlertsBadge count={0} />);
		expect(screen.getByRole('button').className).toContain('noc-c2-fired-quiet');
	});
});
