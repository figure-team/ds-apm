import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Input, Radio } from 'antd';

import { Runbook, RunbookStatus } from './types';

interface Props {
	initial?: Partial<Runbook>;
	onSubmit: (values: Partial<Runbook>) => void;
	onCancel: () => void;
	saving?: boolean;
}

function RunbookForm({ initial, onSubmit, onCancel, saving }: Props): JSX.Element {
	const { t } = useTranslation(['runbooks']);
	const [title, setTitle] = useState(initial?.title ?? '');
	const [description, setDescription] = useState(initial?.description ?? '');
	const [script, setScript] = useState(initial?.executableScript ?? '');
	const [status, setStatus] = useState<RunbookStatus>(
		(initial?.status as RunbookStatus) ?? 'approved',
	);

	const handleSave = (): void => {
		onSubmit({
			title,
			description,
			executableScript: script,
			status,
		});
	};

	return (
		<div className="runbook-form">
			<label>
				{t('field_title')}
				<Input value={title} onChange={(e): void => setTitle(e.target.value)} />
			</label>
			<label>
				{t('field_description')}
				<Input.TextArea
					value={description}
					onChange={(e): void => setDescription(e.target.value)}
					autoSize={{ minRows: 3, maxRows: 12 }}
				/>
			</label>
			<label>
				{t('field_script')}
				<Input.TextArea
					value={script}
					onChange={(e): void => setScript(e.target.value)}
					autoSize={{ minRows: 6, maxRows: 30 }}
					style={{ fontFamily: 'monospace' }}
				/>
			</label>
			<Radio.Group value={status} onChange={(e): void => setStatus(e.target.value)}>
				<Radio value="draft">{t('status_draft')}</Radio>
				<Radio value="approved">{t('status_approved')}</Radio>
			</Radio.Group>
			<div className="runbook-form__actions">
				<Button onClick={onCancel}>{t('btn_cancel')}</Button>
				<Button type="primary" onClick={handleSave} loading={saving}>
					{t('btn_save')}
				</Button>
			</div>
		</div>
	);
}

export default RunbookForm;
