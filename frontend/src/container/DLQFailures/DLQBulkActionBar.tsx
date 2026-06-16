import { Button, Space, Typography } from 'antd';

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
			<Typography.Text>{selectedCount}개 선택됨</Typography.Text>
			<Space>
				<Button type="primary" loading={loading} onClick={onReplay}>
					↩ 재전송
				</Button>
				<Button onClick={onClear}>선택 해제</Button>
			</Space>
		</div>
	);
}
