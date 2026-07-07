import { fireEvent, render, screen } from '@testing-library/react';

import { NocServiceRow } from '../../types';
import AnomalyBadge from '../AnomalyBadge';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate: jest.fn() }),
}));

const SERVICES: NocServiceRow[] = [
	{ name: 'ad', health: 'critical', errPct: 100, p99Ms: 601067, rps: 0 },
	{
		name: 'recommendation',
		health: 'critical',
		errPct: 100,
		p99Ms: 600219,
		rps: 0,
	},
];

describe('AnomalyBadge', () => {
	it('renders nothing when there are no anomalous services', () => {
		const { container } = render(
			<AnomalyBadge services={[]} count={0} overflowCount={0} />,
		);
		expect(container).toBeEmptyDOMElement();
	});

	it('shows badge and opens popover with anomaly cards on click', () => {
		render(<AnomalyBadge services={SERVICES} count={2} overflowCount={0} />);
		const badge = screen.getByRole('button', { name: /noc_c2_anom_badge/ });
		fireEvent.click(badge);
		expect(screen.getByText('noc_c2_watch_anomaly')).toBeInTheDocument();
		expect(screen.getByText('ad')).toBeInTheDocument();
		expect(screen.getByText('recommendation')).toBeInTheDocument();
	});
});
