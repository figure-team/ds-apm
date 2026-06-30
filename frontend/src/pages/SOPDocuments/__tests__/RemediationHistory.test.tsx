import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import RemediationHistory from '../RemediationHistory';

jest.mock('api/remediation', () => ({
	listRemediations: jest.fn(),
}));

const api = require('api/remediation');

describe('RemediationHistory', () => {
	beforeEach(() => jest.clearAllMocks());

	it('renders rows from listRemediations', async () => {
		api.listRemediations.mockResolvedValue([
			{ id: 'r1', status: 'succeeded', scriptSnapshot: 's', sopId: 'SOP-A', runbookId: 'rb', proposedAt: '2026-06-24T00:00:00Z', approvedBy: 'alice', exitCode: 0 },
		]);
		render(<RemediationHistory />);
		expect(await screen.findByText('SOP-A')).toBeInTheDocument();
		expect(screen.getByText('alice')).toBeInTheDocument();
		await waitFor(() => expect(api.listRemediations).toHaveBeenCalled());
	});

	it('shows empty state when no rows', async () => {
		api.listRemediations.mockResolvedValue([]);
		render(<RemediationHistory />);
		expect(await screen.findByText('history_empty')).toBeInTheDocument();
	});

	it('shows error message when listRemediations rejects', async () => {
		api.listRemediations.mockRejectedValue(new Error('network error'));
		render(<RemediationHistory />);
		expect(await screen.findByText('history_load_error')).toBeInTheDocument();
	});

	it('opens the approval popup for a proposed row', async () => {
		api.listRemediations.mockResolvedValue([
			{ id: 'r-prop', status: 'proposed', scriptSnapshot: 's', sopId: 'SOP-P', runbookId: 'rb', proposedAt: '2026-06-24T00:00:00Z' },
		]);
		const openSpy = jest.spyOn(window, 'open').mockReturnValue(null);
		render(<RemediationHistory />);
		fireEvent.click(await screen.findByText('history_action_approve'));
		expect(openSpy).toHaveBeenCalledWith(
			'/remediation/approve/r-prop',
			'_blank',
			expect.stringContaining('noopener'),
		);
		openSpy.mockRestore();
	});

	it('does not show the approve action for a terminal row', async () => {
		api.listRemediations.mockResolvedValue([
			{ id: 'r1', status: 'succeeded', scriptSnapshot: 's', sopId: 'SOP-A', runbookId: 'rb', proposedAt: '2026-06-24T00:00:00Z', approvedBy: 'alice', exitCode: 0 },
		]);
		render(<RemediationHistory />);
		expect(await screen.findByText('SOP-A')).toBeInTheDocument();
		expect(screen.queryByText('history_action_approve')).not.toBeInTheDocument();
	});
});
