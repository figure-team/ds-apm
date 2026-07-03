import { render, screen } from '@testing-library/react';

import { NocAlert } from '../../types';
import AlertsPanel from '../AlertsPanel';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate: jest.fn() }),
}));

const alerts: NocAlert[] = [
	{ id: '1', severity: 'critical', title: 'DB down', meta: 'db', age: '5m' },
];

describe('AlertsPanel', () => {
	it('renders firing alerts', () => {
		render(<AlertsPanel alerts={alerts} isLoading={false} isError={false} />);
		expect(screen.getByText('DB down')).toBeInTheDocument();
	});

	it('empty state shows resolved history', () => {
		render(
			<AlertsPanel
				alerts={[]}
				isLoading={false}
				isError={false}
				lastResolved={{ age: '10m', service: 'cart' }}
			/>,
		);
		expect(screen.getByText('noc_c2_alerts_empty')).toBeInTheDocument();
	});
});
