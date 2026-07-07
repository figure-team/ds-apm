import { fireEvent, render, screen } from '@testing-library/react';

import { NocInfraHost } from '../../types';
import InfraBadge from '../InfraBadge';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

const HOSTS: NocInfraHost[] = [
	{ name: 'node-01', cpu: 42, mem: 55, health: 'healthy' },
	{ name: 'node-03', cpu: 78, mem: 61, health: 'warning' },
];

describe('InfraBadge', () => {
	it('shows host count and warn tone when a host is warning', () => {
		render(<InfraBadge hosts={HOSTS} isLoading={false} isError={false} />);
		const btn = screen.getByRole('button', { name: /noc_c2_infra_badge/ });
		expect(btn.className).toContain('noc-c2-infra-warn');
	});

	it('opens popover with host tiles on click', async () => {
		render(<InfraBadge hosts={HOSTS} isLoading={false} isError={false} />);
		fireEvent.click(screen.getByRole('button', { name: /noc_c2_infra_badge/ }));
		expect(await screen.findByText('node-01')).toBeInTheDocument();
		expect(await screen.findByText('node-03')).toBeInTheDocument();
	});

	it('uses calm tone when all hosts healthy', () => {
		render(<InfraBadge hosts={[HOSTS[0]]} isLoading={false} isError={false} />);
		expect(screen.getByRole('button').className).toContain('noc-c2-infra-calm');
	});
});
