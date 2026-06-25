import { useState } from 'react';
import { useTranslation } from 'react-i18next';
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
	const { t } = useTranslation(['runbooks']);
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
			toast.error(t('toast_need_error_example'));
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
				toast.error(data.error ?? t('toast_drafter_failed'));
			} else {
				setDraft(data as Runbook);
			}
		} catch (error) {
			toast.error(t('toast_draft_error'));
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
			toast.success(t('toast_saved'));
			// Reset state
			setDraft(null);
			setErrorExamples(['', '', '']);
			setAuthIssue(null);
			onSaved();
		} catch (error) {
			toast.error(t('toast_save_error'));
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
			title={t('menu_ai_draft')}
			onCancel={handleCancel}
			footer={null}
			width={720}
			destroyOnClose
		>
			{!draft && (
				<Space direction="vertical" style={{ width: '100%' }} size="middle">
					<p>{t('draft_intro')}</p>

					{authIssue === 'auth' && (
						<Alert
							type="warning"
							showIcon
							message={t('draft_auth_title')}
							description={t('draft_auth_desc')}
						/>
					)}

					{[0, 1, 2].map((index) => (
						<label key={index}>
							{t('draft_error_example')} {index + 1}
							<Input.TextArea
								value={errorExamples[index]}
								onChange={(e): void => handleErrorChange(index, e.target.value)}
								disabled={drafting}
								autoSize={{ minRows: 2, maxRows: 8 }}
								placeholder={
									index === 0
										? t('draft_placeholder_first')
										: t('draft_placeholder_optional')
								}
							/>
						</label>
					))}

					<Space>
						<Button onClick={handleCancel} disabled={drafting}>
							{t('btn_cancel')}
						</Button>
						<Button type="primary" onClick={handleDraft} loading={drafting}>
							{t('btn_draft')}
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
