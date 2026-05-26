import { render, screen, within } from 'tests/test-utils';
import { toast } from '@signozhq/ui';
import { Runbook } from './types';
import RunbookCard from './RunbookCard';

jest.mock('@signozhq/ui', () => ({
	...jest.requireActual('@signozhq/ui'),
	toast: {
		success: jest.fn(),
		error: jest.fn(),
	},
}));

const mockRunbook: Runbook = {
	id: 'rb-1',
	title: 'Restart Service',
	description: 'Restarts the affected microservice to resolve transient issues.',
	executableScript: '#!/bin/bash\necho "Restarting service"\nkubectl restart deployment',
	status: 'approved',
	confidence: 0.95,
	aiDraftedBy: 'claude-ai',
	sourceErrorExamples: ['Connection timeout', 'Service unavailable'],
	createdAt: '2024-01-01T00:00:00Z',
	updatedAt: '2024-01-02T00:00:00Z',
	updatedBy: 'admin',
};

describe('RunbookCard', () => {
	const mockOnEdit = jest.fn();
	const mockOnStatusChange = jest.fn();
	const mockOnDelete = jest.fn();

	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('renders title, status badge, and script preview', () => {
		render(
			<RunbookCard
				runbook={mockRunbook}
				canEdit
				canDelete
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		expect(screen.getByText('Restart Service')).toBeInTheDocument();
		expect(screen.getByText('approved')).toBeInTheDocument();
		expect(screen.getByText(/Restarting service/)).toBeInTheDocument();
	});

	it('copies the script to clipboard on Copy click and calls toast success', async () => {
		const clipboardWriteFn = jest.fn().mockResolvedValue(undefined);
		Object.assign(navigator, {
			clipboard: {
				writeText: clipboardWriteFn,
			},
		});

		render(
			<RunbookCard
				runbook={mockRunbook}
				canEdit
				canDelete
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		const copyButton = screen.getByRole('button', { name: /copy/i });
		copyButton.click();

		expect(clipboardWriteFn).toHaveBeenCalledWith(mockRunbook.executableScript);
		// Wait for promise and toast
		await new Promise((resolve) => setTimeout(resolve, 10));
		expect(toast.success).toHaveBeenCalled();
	});

	it('hides Delete button when canDelete is false', () => {
		render(
			<RunbookCard
				runbook={mockRunbook}
				canEdit
				canDelete={false}
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		expect(screen.queryByRole('button', { name: /delete/i })).not.toBeInTheDocument();
	});

	it('renders AI draft metadata when aiDraftedBy is non-empty', () => {
		render(
			<RunbookCard
				runbook={mockRunbook}
				canEdit
				canDelete
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		expect(screen.getByText(/AI-drafted by/)).toBeInTheDocument();
		expect(screen.getByText(/claude-ai/)).toBeInTheDocument();
		expect(screen.getByText(/95%/)).toBeInTheDocument();
	});

	it('does not render AI draft metadata when aiDraftedBy is empty', () => {
		const runbookWithoutAIDraft = { ...mockRunbook, aiDraftedBy: '' };
		render(
			<RunbookCard
				runbook={runbookWithoutAIDraft}
				canEdit
				canDelete
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		expect(screen.queryByText(/AI-drafted by/)).not.toBeInTheDocument();
	});

	it('shows Edit button only when canEdit is true', () => {
		render(
			<RunbookCard
				runbook={mockRunbook}
				canEdit
				canDelete
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument();
	});

	it('hides Edit button when canEdit is false', () => {
		render(
			<RunbookCard
				runbook={mockRunbook}
				canEdit={false}
				canDelete
				onEdit={mockOnEdit}
				onStatusChange={mockOnStatusChange}
				onDelete={mockOnDelete}
			/>
		);

		expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument();
	});
});
