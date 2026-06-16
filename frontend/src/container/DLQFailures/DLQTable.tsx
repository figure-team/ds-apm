import type { TableProps } from 'antd';
import { Button, Table, Tag, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { DLQEntry, DLQStatus } from 'types/api/dlq';

const STATUS_COLOR: Record<DLQStatus, string> = {
	pending: 'green',
	replayed: 'blue',
	replay_failed: 'red',
};

const STATUS_LABEL: Record<DLQStatus, string> = {
	pending: '대기중',
	replayed: '재전송됨',
	replay_failed: '재전송 실패',
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
			title: '채널',
			dataIndex: 'channel',
			key: 'channel',
			render: (ch: string): JSX.Element => <Tag>{ch}</Tag>,
		},
		{
			title: '실패 이유',
			dataIndex: 'reason',
			key: 'reason',
			ellipsis: true,
		},
		{
			title: '실패 시각',
			dataIndex: 'failed_at',
			key: 'failed_at',
			render: (t: string): string => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
		},
		{
			title: '상태',
			dataIndex: 'status',
			key: 'status',
			render: (s: DLQStatus): JSX.Element => (
				<Tag color={STATUS_COLOR[s]}>{STATUS_LABEL[s]}</Tag>
			),
		},
		{
			title: '페이로드',
			key: 'payload',
			render: (_: unknown, record: DLQEntry): JSX.Element => (
				<Button size="small" onClick={(): void => onViewPayload(record.payload)}>
					보기
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
