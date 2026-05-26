import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import RunbookDraftFromError from './RunbookDraftFromError';

const mockDraft = jest.fn();
const mockCreate = jest.fn();

jest.mock('api/runbook/draftRunbook', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockDraft(...args),
}));
jest.mock('api/runbook/createRunbook', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockCreate(...args),
}));

beforeEach(() => jest.clearAllMocks());

it('calls draftRunbook with pasted error and prefills form on success', async () => {
	mockDraft.mockResolvedValue({
		data: {
			id: '01928374-5566-77ab-89cd-eeff00112233',
			title: 'Restart',
			description: 'x',
			executableScript: '#!/bin/bash\necho hi\n',
			status: 'draft',
			confidence: 0.7,
			aiDraftedBy: 'mock',
			sourceErrorExamples: ['timeout'],
			createdAt: '2026-05-22T00:00:00Z',
			updatedAt: '2026-05-22T00:00:00Z',
			updatedBy: 'ai',
		},
	});
	render(
		<RunbookDraftFromError
			sopId="SOP-PAY-001"
			version="v01"
			open
			onSaved={jest.fn()}
			onCancel={jest.fn()}
		/>,
	);
	fireEvent.change(screen.getByLabelText(/error example 1/i), {
		target: { value: 'timeout' },
	});
	fireEvent.click(screen.getByRole('button', { name: /^draft$/i }));
	await waitFor(() => expect(mockDraft).toHaveBeenCalled());
	await waitFor(() => expect(screen.getByDisplayValue('Restart')).toBeInTheDocument());
});

it('shows auth banner when API returns errorKind=auth', async () => {
	mockDraft.mockResolvedValue({
		data: { ok: false, error: 'auth failed', errorKind: 'auth' },
	});
	render(
		<RunbookDraftFromError
			sopId="SOP-PAY-001"
			version="v01"
			open
			onSaved={jest.fn()}
			onCancel={jest.fn()}
		/>,
	);
	fireEvent.change(screen.getByLabelText(/error example 1/i), {
		target: { value: 'something' },
	});
	fireEvent.click(screen.getByRole('button', { name: /^draft$/i }));
	await waitFor(() =>
		expect(screen.getByText(/authentication issue/i)).toBeInTheDocument(),
	);
});
