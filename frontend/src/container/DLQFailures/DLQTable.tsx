import type { TableProps } from 'antd';
import { Button, Table, Tag, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { DLQEntry, DLQStatus } from 'types/api/dlq';

const STATUS_COLOR: Record<DLQStatus, string> = {
	pending: 'green',
	replayed: 'blue',
	replay_failed: 'red',
};

const STATUS_LABEL_KEY: Record<DLQStatus, string> = {
	pending: 'dlq_status_pending',
	replayed: 'dlq_status_replayed',
	replay_failed: 'dlq_status_replay_failed',
};

interface DLQTableProps {
	entries: DLQEntry[];
	selectedKeys: string[];
	onSelectionChange: (keys: string[]) => void;
	onViewPayload: (payload: string) => void;
}

export function DLQTable({
	entries,
	selectedKeys,
	onSelectionChange,
	onViewPayload,
}: DLQTableProps): JSX.Element {
	const { t } = useTranslation(['channels']);
	const columns: TableProps<DLQEntry>['columns'] = [
		{
			title: 'Alert ID',
			dataIndex: 'event_id',
			key: 'event_id',
			render: (id: string): JSX.Element => (
				<Tooltip title={id}>
					<code>{id.slice(0, 12)}</code>
				</Tooltip>
			),
		},
		{
			title: t('dlq_column_channel'),
			dataIndex: 'channel',
			key: 'channel',
			render: (ch: string): JSX.Element => <Tag>{ch}</Tag>,
		},
		{
			title: t('dlq_column_reason'),
			dataIndex: 'reason',
			key: 'reason',
			ellipsis: true,
		},
		{
			title: t('dlq_column_failed_at'),
			dataIndex: 'failed_at',
			key: 'failed_at',
			render: (t: string): string => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
		},
		{
			title: t('dlq_column_status'),
			dataIndex: 'status',
			key: 'status',
			render: (s: DLQStatus): JSX.Element => (
				<Tag color={STATUS_COLOR[s]}>{t(STATUS_LABEL_KEY[s])}</Tag>
			),
		},
		{
			title: t('dlq_column_payload'),
			key: 'payload',
			render: (_: unknown, record: DLQEntry): JSX.Element => (
				<Button size="small" onClick={(): void => onViewPayload(record.payload)}>
					{t('dlq_btn_view')}
				</Button>
			),
		},
	];

	return (
		<Table<DLQEntry>
			rowKey="event_id"
			columns={columns}
			dataSource={entries}
			rowSelection={{
				selectedRowKeys: selectedKeys,
				onChange: (keys): void => onSelectionChange(keys as string[]),
			}}
			pagination={{ pageSize: 20 }}
		/>
	);
}
