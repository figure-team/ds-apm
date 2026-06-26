import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import RemediationCard from '../index';

jest.mock('api/remediation', () => ({
	getRemediation: jest.fn().mockResolvedValue({
		id: 'rem-1',
		status: 'proposed',
		scriptSnapshot: '#!/bin/bash\necho hi',
		sopId: 'SOP-1',
		runbookId: 'rb-1',
	}),
	approveRemediation: jest.fn().mockResolvedValue({ id: 'rem-1', status: 'executing', scriptSnapshot: '', sopId: 'SOP-1', runbookId: 'rb-1' }),
	rejectRemediation: jest.fn().mockResolvedValue({ id: 'rem-1', status: 'rejected', scriptSnapshot: '', sopId: 'SOP-1', runbookId: 'rb-1' }),
}));

describe('RemediationCard', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		// Suppress window.confirm noise in tests
		window.confirm = jest.fn().mockReturnValue(true);
	});

	it('renders proposed state with approve/reject buttons', async () => {
		render(<RemediationCard remediationId="rem-1" />);

		// i18n mock returns keys
		expect(await screen.findByText('remediation_approve')).toBeInTheDocument();
		expect(screen.getByText('remediation_reject')).toBeInTheDocument();
		expect(screen.getByText(/echo hi/)).toBeInTheDocument();
	});

	it('renders the script snapshot content', async () => {
		render(<RemediationCard remediationId="rem-1" />);

		expect(await screen.findByText(/echo hi/)).toBeInTheDocument();
	});

	it('calls approveRemediation on approve click', async () => {
		const api = require('api/remediation');
		render(<RemediationCard remediationId="rem-1" />);

		fireEvent.click(await screen.findByText('remediation_approve'));

		await waitFor(() =>
			expect(api.approveRemediation).toHaveBeenCalledWith('rem-1'),
		);
	});

	it('leads with the execution result and tucks the script behind a toggle when run', async () => {
		const api = require('api/remediation');
		api.getRemediation.mockResolvedValueOnce({
			id: 'rem-1',
			status: 'failed',
			scriptSnapshot: 'echo ran',
			sopId: 'SOP-1',
			runbookId: 'rb-1',
			exitCode: 1,
			outputSnippet: 'boom',
		});

		render(<RemediationCard remediationId="rem-1" />);

		// Result is shown; card title switches to the result variant.
		expect(await screen.findByText('boom')).toBeInTheDocument();
		expect(screen.getByText('remediation_result_card_title')).toBeInTheDocument();
		// Script is available behind the toggle (not the primary content).
		expect(screen.getByText('remediation_show_script')).toBeInTheDocument();
		expect(screen.getByText(/echo ran/)).toBeInTheDocument();
	});

	it('does not show action buttons for terminal status', async () => {
		const api = require('api/remediation');
		api.getRemediation.mockResolvedValueOnce({
			id: 'rem-1',
			status: 'verified',
			scriptSnapshot: 'echo done',
			sopId: 'SOP-1',
			runbookId: 'rb-1',
		});

		render(<RemediationCard remediationId="rem-1" />);

		await screen.findByText(/echo done/);

		expect(screen.queryByText('remediation_approve')).not.toBeInTheDocument();
		expect(screen.queryByText('remediation_reject')).not.toBeInTheDocument();
	});
});
