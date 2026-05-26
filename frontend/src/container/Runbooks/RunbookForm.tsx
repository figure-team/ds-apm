import { useState } from 'react';
import { Button, Input, Radio } from 'antd';

import { Runbook, RunbookStatus } from './types';

interface Props {
	initial?: Partial<Runbook>;
	onSubmit: (values: Partial<Runbook>) => void;
	onCancel: () => void;
	saving?: boolean;
}

function RunbookForm({ initial, onSubmit, onCancel, saving }: Props): JSX.Element {
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
				Title
				<Input value={title} onChange={(e): void => setTitle(e.target.value)} />
			</label>
			<label>
				Description (markdown)
				<Input.TextArea
					value={description}
					onChange={(e): void => setDescription(e.target.value)}
					autoSize={{ minRows: 3, maxRows: 12 }}
				/>
			</label>
			<label>
				Script (bash)
				<Input.TextArea
					value={script}
					onChange={(e): void => setScript(e.target.value)}
					autoSize={{ minRows: 6, maxRows: 30 }}
					style={{ fontFamily: 'monospace' }}
				/>
			</label>
			<Radio.Group value={status} onChange={(e): void => setStatus(e.target.value)}>
				<Radio value="draft">Draft</Radio>
				<Radio value="approved">Approved</Radio>
			</Radio.Group>
			<div className="runbook-form__actions">
				<Button onClick={onCancel}>Cancel</Button>
				<Button type="primary" onClick={handleSave} loading={saving}>
					Save
				</Button>
			</div>
		</div>
	);
}

export default RunbookForm;
