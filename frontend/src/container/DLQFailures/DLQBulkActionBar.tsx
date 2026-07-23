import { Button, Space, Typography } from 'antd';
import { useTranslation } from 'react-i18next';

interface DLQBulkActionBarProps {
	selectedCount: number;
	loading: boolean;
	onReplay: () => void;
	onClear: () => void;
}

export function DLQBulkActionBar({
	selectedCount,
	loading,
	onReplay,
	onClear,
}: DLQBulkActionBarProps): JSX.Element | null {
	const { t } = useTranslation(['channels']);

	if (selectedCount === 0) return null;

	return (
		<div
			style={{
				padding: '8px 16px',
				background: 'var(--bg-slate-400)',
				borderRadius: 6,
				marginBottom: 8,
				display: 'flex',
				alignItems: 'center',
				gap: 12,
			}}
		>
			<Typography.Text>
				{t('dlq_selected_count', { selected: selectedCount })}
			</Typography.Text>
			<Space>
				<Button type="primary" loading={loading} onClick={onReplay}>
					{t('dlq_btn_replay')}
				</Button>
				<Button onClick={onClear}>{t('dlq_btn_clear_selection')}</Button>
			</Space>
		</div>
	);
}
