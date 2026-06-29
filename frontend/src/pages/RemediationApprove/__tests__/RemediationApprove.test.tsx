import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import RemediationApprove from '../index';

jest.mock('react-router-dom', () => ({
	...jest.requireActual('react-router-dom'),
	useParams: (): { id: string } => ({ id: 'rem-1' }),
}));

jest.mock('api/remediation', () => ({
	getRemediation: jest.fn(),
	approveRemediation: jest.fn(),
	rejectRemediation: jest.fn(),
}));

const api = require('api/remediation');

describe('RemediationApprove', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		window.confirm = jest.fn().mockReturnValue(true);
	});

	it('shows approve/reject for a proposed remediation', async () => {
		api.getRemediation.mockResolvedValue({
			id: 'rem-1', status: 'proposed', scriptSnapshot: 'echo hi', sopId: 'SOP-1', runbookId: 'rb-1',
		});
		render(<RemediationApprove />);
		expect(await screen.findByText('remediation_approve')).toBeInTheDocument();
		expect(screen.getByText('remediation_reject')).toBeInTheDocument();
	});

	it('shows completion screen after approve', async () => {
		api.getRemediation.mockResolvedValue({
			id: 'rem-1', status: 'proposed', scriptSnapshot: 'echo hi', sopId: 'SOP-1', runbookId: 'rb-1',
		});
		api.approveRemediation.mockResolvedValue({ id: 'rem-1', status: 'executing', scriptSnapshot: '', sopId: 'SOP-1', runbookId: 'rb-1' });
		render(<RemediationApprove />);
		fireEvent.click(await screen.findByText('remediation_approve'));
		expect(await screen.findByText('remediation_approved_done_title')).toBeInTheDocument();
		await waitFor(() => expect(api.approveRemediation).toHaveBeenCalledWith('rem-1'));
	});

	it('shows expired screen for expired remediation', async () => {
		api.getRemediation.mockResolvedValue({
			id: 'rem-1', status: 'expired', scriptSnapshot: '', sopId: 'SOP-1', runbookId: 'rb-1',
		});
		render(<RemediationApprove />);
		expect(await screen.findByText('remediation_expired_title')).toBeInTheDocument();
		expect(screen.queryByText('remediation_approve')).not.toBeInTheDocument();
	});

	it('shows not-found message when get fails', async () => {
		api.getRemediation.mockRejectedValue(new Error('404'));
		render(<RemediationApprove />);
		expect(await screen.findByText('remediation_not_found')).toBeInTheDocument();
	});
});
