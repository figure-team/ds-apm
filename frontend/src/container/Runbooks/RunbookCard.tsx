import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { Button, Tag } from 'antd';
import { CopyOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { toast } from '@signozhq/ui';
import { Runbook, RunbookStatus } from './types';
import './Runbooks.styles.scss';

interface Props {
	runbook: Runbook;
	canEdit: boolean;
	canDelete: boolean;
	onEdit: (rb: Runbook) => void;
	onStatusChange: (rb: Runbook, next: RunbookStatus) => void;
	onDelete: (rb: Runbook) => void;
}

export default function RunbookCard({
	runbook,
	canEdit,
	canDelete,
	onEdit,
	onStatusChange,
	onDelete,
}: Props): JSX.Element {
	const { t } = useTranslation(['runbooks']);
	const isDarkMode = useIsDarkMode();
	const STATUS_COLORS: Record<RunbookStatus, string> = {
		draft: isDarkMode ? 'gold' : '#D97706',
		approved: isDarkMode ? 'green' : '#16A34A',
		deprecated: 'default',
	};

	const handleCopyScript = useCallback(async () => {
		try {
			await navigator.clipboard.writeText(runbook.executableScript);
			const lines = runbook.executableScript.split('\n').length;
			toast.success(t('toast_script_copied', { lines }));
		} catch (error) {
			toast.error(t('toast_copy_error'));
		}
	}, [runbook.executableScript, t]);

	const handleStatusToggle = useCallback(() => {
		const nextStatus: RunbookStatus =
			runbook.status === 'approved' ? 'deprecated' : 'approved';
		onStatusChange(runbook, nextStatus);
	}, [runbook, onStatusChange]);

	const handleDelete = useCallback(() => {
		if (window.confirm(t('confirm_delete', { title: runbook.title }))) {
			onDelete(runbook);
		}
	}, [runbook, onDelete, t]);

	const truncatedDescription =
		runbook.description.length > 280
			? `${runbook.description.substring(0, 280)}...`
			: runbook.description;

	return (
		<div className="runbook-card">
			<div className="runbook-card__header">
				<h3>{runbook.title}</h3>
				<Tag color={STATUS_COLORS[runbook.status]}>
					{t(`status_${runbook.status}`)}
				</Tag>
			</div>

			{runbook.aiDraftedBy && (
				<div className="runbook-card__meta">
					{t('card_ai_drafted_by')} <code>{runbook.aiDraftedBy}</code> ·{' '}
					{t('card_confidence', {
						pct: (runbook.confidence * 100).toFixed(0),
					})}
				</div>
			)}

			<p className="runbook-card__description">{truncatedDescription}</p>

			<div className="runbook-card__script">
				<pre>
					<code>{runbook.executableScript}</code>
				</pre>
			</div>

			<div className="runbook-card__actions">
				<Button
					type="primary"
					icon={<CopyOutlined />}
					onClick={handleCopyScript}
				>
					{t('btn_copy')}
				</Button>

				{canEdit && (
					<>
						<Button icon={<EditOutlined />} onClick={() => onEdit(runbook)}>
							{t('btn_edit')}
						</Button>
						<Button onClick={handleStatusToggle}>
							{runbook.status === 'approved' ? t('btn_deprecate') : t('btn_approve')}
						</Button>
					</>
				)}

				{canDelete && (
					<Button
						type="primary"
						danger
						icon={<DeleteOutlined />}
						onClick={handleDelete}
					>
						{t('btn_delete')}
					</Button>
				)}
			</div>
		</div>
	);
}
