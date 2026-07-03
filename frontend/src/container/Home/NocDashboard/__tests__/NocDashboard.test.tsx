import { render, screen } from '@testing-library/react';

import NocDashboard from '../NocDashboard';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useDarkMode', () => ({ useIsDarkMode: () => true }));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate: jest.fn() }),
}));
jest.mock('container/TopNav/DateTimeSelectionV2', () => (): JSX.Element => (
	<div data-testid="dt" />
));
jest.mock('../hooks/useNocAlerts', () => ({
	__esModule: true,
	default: () => ({
		alerts: [],
		firingCount: 0,
		totalCount: 0,
		isLoading: false,
		isError: false,
	}),
}));
jest.mock('../hooks/useNocOverview', () => ({
	__esModule: true,
	default: () => ({ services: [], kpis: [], isLoading: false, isError: false }),
}));
jest.mock('../hooks/useNocTrend', () => ({
	__esModule: true,
	default: () => ({ series: [], stepSec: 60, isLoading: false, isError: false }),
}));
jest.mock('../hooks/useNocInfra', () => ({
	__esModule: true,
	default: () => ({ hosts: [], isLoading: false, isError: false }),
}));

describe('NocDashboard (C-2)', () => {
	it('assembles summary band and panels without crashing', () => {
		render(<NocDashboard />);
		// summary band title
		expect(screen.getByText('noc_c2_title')).toBeInTheDocument();
		// healthy empty state (all counts zero -> stable pill)
		expect(screen.getByText('noc_c2_stable_title')).toBeInTheDocument();
	});
});
