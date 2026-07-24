import { fireEvent, render, screen } from '@testing-library/react';

import dashboardUrl from '@/assets/Icons/dashboard.svg';
import wrenchUrl from '@/assets/Icons/wrench.svg';

import ExplorerActionCard from '../ExplorerActionCard';

const mockSafeNavigate = jest.fn();
const mockLogEvent = jest.fn();

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: (...args: unknown[]): void => mockLogEvent(...args),
}));

jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: (): { safeNavigate: jest.Mock } => ({
		safeNavigate: mockSafeNavigate,
	}),
}));

const ACTIONS = [
	{
		label: 'Open logs explorer',
		icon: <span />,
		source: 'Logs',
		route: '/logs',
	},
	{
		label: 'Open traces explorer',
		icon: <span />,
		source: 'Traces',
		route: '/traces',
	},
];

describe('ExplorerActionCard', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('renders title, description and one button per action', () => {
		render(
			<ExplorerActionCard
				iconUrl={wrenchUrl}
				iconAlt="wrench"
				title="Explorers"
				description="Dig into your data"
				actions={ACTIONS}
			/>,
		);
		expect(screen.getByText('Explorers')).toBeInTheDocument();
		expect(screen.getByText('Dig into your data')).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Open logs explorer' }),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Open traces explorer' }),
		).toBeInTheDocument();
	});

	it('logs and navigates with the action-specific source and route', () => {
		render(
			<ExplorerActionCard
				iconUrl={wrenchUrl}
				iconAlt="wrench"
				title="Explorers"
				description="Dig into your data"
				actions={ACTIONS}
			/>,
		);
		fireEvent.click(screen.getByRole('button', { name: 'Open traces explorer' }));
		expect(mockLogEvent).toHaveBeenCalledWith('Homepage: Explore clicked', {
			source: 'Traces',
		});
		expect(mockSafeNavigate).toHaveBeenCalledWith('/traces', { newTab: false });
	});

	it('opens a new tab when a modifier key is held', () => {
		render(
			<ExplorerActionCard
				iconUrl={wrenchUrl}
				iconAlt="wrench"
				title="Explorers"
				description="Dig into your data"
				actions={ACTIONS}
			/>,
		);
		fireEvent.click(screen.getByRole('button', { name: 'Open logs explorer' }), {
			metaKey: true,
		});
		expect(mockSafeNavigate).toHaveBeenCalledWith('/logs', { newTab: true });
	});

	it('marks the icon lazy only when lazyIcon is set', () => {
		const { rerender } = render(
			<ExplorerActionCard
				iconUrl={wrenchUrl}
				iconAlt="wrench"
				title="Explorers"
				description="d"
				actions={[]}
				lazyIcon
			/>,
		);
		expect(screen.getByAltText('wrench')).toHaveAttribute('loading', 'lazy');

		rerender(
			<ExplorerActionCard
				iconUrl={dashboardUrl}
				iconAlt="dashboard"
				title="Dashboards"
				description="d"
				actions={[]}
			/>,
		);
		expect(screen.getByAltText('dashboard')).not.toHaveAttribute('loading');
	});
});
