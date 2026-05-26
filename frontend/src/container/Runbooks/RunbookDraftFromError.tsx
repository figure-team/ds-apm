import { useState } from 'react';
import { Alert, Button, Input, Modal, Space } from 'antd';
import { toast } from '@signozhq/ui';
import createRunbook from 'api/runbook/createRunbook';
import draftRunbook from 'api/runbook/draftRunbook';

import { Runbook, RunbookErrorKind } from './types';
import RunbookForm from './RunbookForm';

interface Props {
	sopId: string;
	version: string;
	open: boolean;
	onSaved: () => void;
	onCancel: () => void;
}

export default function RunbookDraftFromError({
	sopId,
	version,
	open,
	onSaved,
	onCancel,
}: Props): JSX.Element {
	const [errorExamples, setErrorExamples] = useState<string[]>(['', '', '']);
	const [drafting, setDrafting] = useState(false);
	const [draft, setDraft] = useState<Runbook | null>(null);
	const [authIssue, setAuthIssue] = useState<RunbookErrorKind | null>(null);
	const [saving, setSaving] = useState(false);

	const handleErrorChange = (index: number, value: string): void => {
		setErrorExamples((prev) => {
			const next = [...prev];
			next[index] = value;
			return next;
		});
	};

	const handleDraft = async (): Promise<void> => {
		const filled = errorExamples.map((e) => e.trim()).filter(Boolean);
		if (filled.length === 0) {
			toast.error('Provide at least one error example');
			return;
		}
		setDrafting(true);
		setAuthIssue(null);
		try {
			const res = await draftRunbook({ sopId, version, errorExamples: filled });
			// draftRunbook returns AxiosResponse<Runbook | DraftRunbookResult>
			const data = res.data;
			if ('ok' in data && data.ok === false) {
				setAuthIssue(data.errorKind ?? 'other');
				toast.error(data.error ?? 'AI drafter failed');
			} else {
				setDraft(data as Runbook);
			}
		} catch (error) {
			toast.error('Failed to draft runbook');
			console.error(error);
		} finally {
			setDrafting(false);
		}
	};

	const handleSave = async (values: Partial<Runbook>): Promise<void> => {
		setSaving(true);
		try {
			await createRunbook(sopId, version, {
				...(draft ?? {}),
				...values,
				sourceErrorExamples: errorExamples.map((e) => e.trim()).filter(Boolean),
			});
			toast.success('Runbook saved');
			// Reset state
			setDraft(null);
			setErrorExamples(['', '', '']);
			setAuthIssue(null);
			onSaved();
		} catch (error) {
			toast.error('Failed to save runbook');
			console.error(error);
		} finally {
			setSaving(false);
		}
	};

	const handleCancel = (): void => {
		setDraft(null);
		setErrorExamples(['', '', '']);
		setAuthIssue(null);
		onCancel();
	};

	return (
		<Modal
			open={open}
			title="AI draft from error"
			onCancel={handleCancel}
			footer={null}
			width={720}
			destroyOnClose
		>
			{!draft && (
				<Space direction="vertical" style={{ width: '100%' }} size="middle">
					<p>Paste up to 3 error examples. The AI will suggest a runbook draft.</p>

					{authIssue === 'auth' && (
						<Alert
							type="warning"
							showIcon
							message="Authentication issue detected"
							description="The LLM provider rejected the request. Check the AI Module Settings."
						/>
					)}

					{[0, 1, 2].map((index) => (
						<label key={index}>
							Error example {index + 1}
							<Input.TextArea
								value={errorExamples[index]}
								onChange={(e): void => handleErrorChange(index, e.target.value)}
								disabled={drafting}
								autoSize={{ minRows: 2, maxRows: 8 }}
								placeholder={index === 0 ? 'paste a recent error message...' : 'optional'}
							/>
						</label>
					))}

					<Space>
						<Button onClick={handleCancel} disabled={drafting}>Cancel</Button>
						<Button type="primary" onClick={handleDraft} loading={drafting}>
							Draft
						</Button>
					</Space>
				</Space>
			)}

			{draft && (
				<RunbookForm
					initial={draft}
					onSubmit={handleSave}
					onCancel={handleCancel}
					saving={saving}
				/>
			)}
		</Modal>
	);
}
