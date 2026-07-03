import { render, screen } from '@testing-library/react';

import { NocInfraHost } from '../../types';
import InfraPanel from '../InfraPanel';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

const hosts: NocInfraHost[] = [
	{ name: 'host-1', cpu: 92, mem: 40, health: 'critical' },
	{ name: 'host-2', cpu: 20, mem: 30, health: 'healthy' },
];

describe('InfraPanel', () => {
	it('renders a tile per host with cpu percent', () => {
		render(<InfraPanel hosts={hosts} isLoading={false} isError={false} />);
		expect(screen.getByText('host-1')).toBeInTheDocument();
		expect(screen.getByText('92%')).toBeInTheDocument();
	});

	it('shows empty message when no hosts', () => {
		render(<InfraPanel hosts={[]} isLoading={false} isError={false} />);
		expect(screen.getByText('noc_c2_infra_empty')).toBeInTheDocument();
	});
});
