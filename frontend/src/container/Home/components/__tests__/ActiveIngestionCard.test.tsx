import { fireEvent, render, screen } from '@testing-library/react';

import ActiveIngestionCard from '../ActiveIngestionCard';

const mockSafeNavigate = jest.fn();
const mockLogEvent = jest.fn();
const mockHistoryPush = jest.fn();

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: (...args: unknown[]): void => mockLogEvent(...args),
}));

jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: (): { safeNavigate: jest.Mock } => ({
		safeNavigate: mockSafeNavigate,
	}),
}));

jest.mock('lib/history', () => ({
	__esModule: true,
	default: { push: (...args: unknown[]): void => mockHistoryPush(...args) },
}));

const PROPS = {
	description: 'Logs are flowing',
	exploreLabel: 'Explore logs',
	source: 'Logs',
	route: '/logs/logs-explorer',
};

describe('ActiveIngestionCard', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('renders the description and explore label', () => {
		render(<ActiveIngestionCard {...PROPS} />);
		expect(screen.getByText('Logs are flowing')).toBeInTheDocument();
		expect(screen.getByText('Explore logs')).toBeInTheDocument();
	});

	it('logs and navigates in the same tab on plain click', () => {
		render(<ActiveIngestionCard {...PROPS} />);
		fireEvent.click(screen.getByRole('button'));
		expect(mockLogEvent).toHaveBeenCalledWith(
			'Homepage: Ingestion Active Explore clicked',
			{ source: 'Logs' },
		);
		expect(mockSafeNavigate).toHaveBeenCalledWith('/logs/logs-explorer', {
			newTab: false,
		});
	});

	it('opens a new tab when a modifier key is held', () => {
		render(<ActiveIngestionCard {...PROPS} />);
		fireEvent.click(screen.getByRole('button'), { ctrlKey: true });
		expect(mockSafeNavigate).toHaveBeenCalledWith('/logs/logs-explorer', {
			newTab: true,
		});
	});

	it('logs and pushes history on Enter key', () => {
		render(<ActiveIngestionCard {...PROPS} />);
		fireEvent.keyDown(screen.getByRole('button'), { key: 'Enter' });
		expect(mockLogEvent).toHaveBeenCalledWith(
			'Homepage: Ingestion Active Explore clicked',
			{ source: 'Logs' },
		);
		expect(mockHistoryPush).toHaveBeenCalledWith('/logs/logs-explorer');
		expect(mockSafeNavigate).not.toHaveBeenCalled();
	});

	it('ignores non-Enter keys', () => {
		render(<ActiveIngestionCard {...PROPS} />);
		fireEvent.keyDown(screen.getByRole('button'), { key: 'a' });
		expect(mockLogEvent).not.toHaveBeenCalled();
		expect(mockHistoryPush).not.toHaveBeenCalled();
	});
});
