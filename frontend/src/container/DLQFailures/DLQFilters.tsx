import { Select, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import { DLQEntry, DLQStatus, GetDLQEntriesParams } from 'types/api/dlq';

interface DLQFiltersProps {
	entries: DLQEntry[];
	filters: GetDLQEntriesParams;
	onChange: (filters: GetDLQEntriesParams) => void;
}

export function DLQFilters({
	entries,
	filters,
	onChange,
}: DLQFiltersProps): JSX.Element {
	const { t } = useTranslation(['channels']);

	const statusOptions = [
		{ value: '', label: t('dlq_filter_all_status') },
		{ value: 'pending', label: t('dlq_status_pending') },
		{ value: 'replayed', label: t('dlq_status_replayed') },
		{ value: 'replay_failed', label: t('dlq_status_replay_failed') },
	];

	const channelOptions = [
		{ value: '', label: t('dlq_filter_all_channels') },
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
				options={statusOptions}
				onChange={(val): void =>
					onChange({ ...filters, status: (val as DLQStatus) || undefined })
				}
			/>
		</Space>
	);
}
