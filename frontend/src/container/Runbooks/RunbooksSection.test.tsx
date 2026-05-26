import { render, screen, waitFor } from 'tests/test-utils';
import { Runbook } from './types';
import RunbooksSection from './RunbooksSection';

const mockRunbook: Runbook = {
	id: 'rb-1',
	title: 'Restart Service',
	description: 'Restarts the affected microservice to resolve transient issues.',
	executableScript: '#!/bin/bash\necho "Restarting service"',
	status: 'approved',
	confidence: 0.95,
	aiDraftedBy: 'claude-ai',
	sourceErrorExamples: ['Connection timeout'],
	createdAt: '2024-01-01T00:00:00Z',
	updatedAt: '2024-01-02T00:00:00Z',
	updatedBy: 'admin',
};

const mockListRunbooks = jest.fn();

jest.mock('api/runbook/listRunbooks', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockListRunbooks(...args),
}));

jest.mock('./RunbookCard', () => {
	return function MockRunbookCard({ runbook }: { runbook: Runbook }) {
		return <div data-testid={`runbook-card-${runbook.id}`}>{runbook.title}</div>;
	};
});

describe('RunbooksSection', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('renders runbooks fetched on mount', async () => {
		mockListRunbooks.mockResolvedValue({
			data: {
				runbooks: [mockRunbook],
			},
		});

		render(<RunbooksSection sopId="sop-1" version="1.0" />);

		await waitFor(() => {
			expect(screen.getByTestId('runbook-card-rb-1')).toBeInTheDocument();
			expect(screen.getByText('Restart Service')).toBeInTheDocument();
		});

		expect(mockListRunbooks).toHaveBeenCalledWith('sop-1', '1.0', 'approved,draft');
	});

	it('shows empty-state copy when API returns no runbooks', async () => {
		mockListRunbooks.mockResolvedValue({
			data: {
				runbooks: [],
			},
		});

		render(<RunbooksSection sopId="sop-1" version="1.0" />);

		await waitFor(() => {
			expect(screen.getByText(/no runbooks yet/i)).toBeInTheDocument();
		});
	});

	it('fetches runbooks with the default status filter on mount', async () => {
		mockListRunbooks.mockResolvedValue({
			data: {
				runbooks: [],
			},
		});

		render(<RunbooksSection sopId="sop-1" version="1.0" />);

		await waitFor(() => {
			expect(mockListRunbooks).toHaveBeenCalledWith('sop-1', '1.0', 'approved,draft');
		});
	});
});
