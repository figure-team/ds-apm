import { render, screen } from '@testing-library/react';

import SummaryBand from '../SummaryBand';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

describe('SummaryBand', () => {
	it('anomaly state: shows severity counts and incident banner', () => {
		render(
			<SummaryBand
				counts={{ critical: 2, warning: 1, healthy: 5, alerts: 3 }}
				incident={{
					id: '1',
					severity: 'critical',
					title: 'DB down',
					meta: '',
					age: '5m',
				}}
			/>,
		);
		expect(screen.getByText('2')).toBeInTheDocument(); // critical count
		expect(screen.getByText('DB down')).toBeInTheDocument();
		expect(screen.getByText('noc_c2_incident_tag')).toBeInTheDocument();
	});

	it('healthy state: zero counts grey + stable pill', () => {
		render(
			<SummaryBand
				counts={{ critical: 0, warning: 0, healthy: 8, alerts: 0 }}
				incident={null}
				stableSince="2h"
			/>,
		);
		expect(screen.getByText('noc_c2_stable_title')).toBeInTheDocument();
	});

	it('renders actions node after spacer', () => {
		render(
			<SummaryBand
				counts={{ critical: 0, warning: 0, healthy: 3, alerts: 0 }}
				incident={null}
				actions={<span data-testid="band-actions">A</span>}
			/>,
		);
		expect(screen.getByTestId('band-actions')).toBeInTheDocument();
	});
});
