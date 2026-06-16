import { Select, Space } from 'antd';
import { DLQEntry, DLQStatus, GetDLQEntriesParams } from 'types/api/dlq';

interface DLQFiltersProps {
	entries: DLQEntry[];
	filters: GetDLQEntriesParams;
	onChange: (filters: GetDLQEntriesParams) => void;
}

const STATUS_OPTIONS = [
	{ value: '', label: '전체 상태' },
	{ value: 'pending', label: '대기중' },
	{ value: 'replayed', label: '재전송됨' },
	{ value: 'replay_failed', label: '재전송 실패' },
];

export function DLQFilters({
	entries,
	filters,
	onChange,
}: DLQFiltersProps): JSX.Element {
	const channelOptions = [
		{ value: '', label: '전체 채널' },
		...Array.from(new Set(entries.map((e) => e.channel))).map((ch) => ({
			value: ch,
			label: ch,
		})),
	];

	return (
		<Space style={{ marginBottom: 16 }}>
			<Select
				style={{ width: 160 }}
				value={filters.channel ?? ''}
				options={channelOptions}
				onChange={(val): void => onChange({ ...filters, channel: val || undefined })}
			/>
			<Select
				style={{ width: 160 }}
				value={filters.status ?? ''}
				options={STATUS_OPTIONS}
				onChange={(val): void =>
					onChange({ ...filters, status: (val as DLQStatus) || undefined })
				}
			/>
		</Space>
	);
}
