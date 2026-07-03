import { render, screen } from '@testing-library/react';

import { NocServiceRow } from '../../types';
import WatchCards from '../WatchCards';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate: jest.fn() }),
}));

const rows: NocServiceRow[] = [
	{ name: 'cart', health: 'critical', p99Ms: 120, errPct: 9, rps: 50 },
	{ name: 'auth', health: 'warning', p99Ms: 80, errPct: 2, rps: 30 },
];

describe('WatchCards', () => {
	it('anomaly mode: renders header key and one card per service', () => {
		render(<WatchCards services={rows} mode="anomaly" />);
		expect(screen.getByText('noc_c2_watch_anomaly')).toBeInTheDocument();
		expect(screen.getByText('cart')).toBeInTheDocument();
		expect(screen.getByText('auth')).toBeInTheDocument();
	});

	it('watch mode: uses watch header', () => {
		render(<WatchCards services={rows} mode="watch" />);
		expect(screen.getByText('noc_c2_watch_normal')).toBeInTheDocument();
	});

	it('shows +N critical affordance when overflowCount > 0', () => {
		render(<WatchCards services={rows} mode="anomaly" overflowCount={3} />);
		expect(screen.getByText('noc_c2_watch_overflow')).toBeInTheDocument();
	});

	it('renders no placeholder cards when fewer than 5', () => {
		render(<WatchCards services={rows} mode="watch" />);
		expect(screen.getAllByRole('button')).toHaveLength(2); // exactly 2, no empty slots
	});
});
