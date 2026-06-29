import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Input, Select, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';

import { listRemediations, RemediationExecution } from 'api/remediation';
import RemediationStatusBadge from 'components/Remediation/RemediationStatusBadge';
import RemediationResult from 'components/Remediation/RemediationResult';

const STATUS_OPTIONS = [
	'proposed', 'executing', 'succeeded', 'failed',
	'verified', 'unresolved', 'rejected', 'expired',
];
const LIST_LIMIT = 200;

function RemediationHistory(): JSX.Element {
	const { t } = useTranslation('sop_documents');
	const [rows, setRows] = useState<RemediationExecution[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [statusFilter, setStatusFilter] = useState<string>('');
	const [sopFilter, setSopFilter] = useState<string>('');
	const [debouncedSop, setDebouncedSop] = useState<string>('');

	// Debounce sopFilter → debouncedSop (300 ms)
	useEffect(() => {
		const timer = setTimeout(() => {
			setDebouncedSop(sopFilter);
		}, 300);
		return (): void => clearTimeout(timer);
	}, [sopFilter]);

	const load = useCallback(async (): Promise<RemediationExecution[]> => {
		const data = await listRemediations({
			status: statusFilter || undefined,
			sopId: debouncedSop || undefined,
			limit: LIST_LIMIT,
		});
		return data;
	}, [statusFilter, debouncedSop]);

	useEffect(() => {
		let active = true;
		setLoading(true);
		setError(null);
		load()
			.then((data) => {
				if (active) setRows(data);
			})
			.catch(() => {
				if (active) setError(t('history_load_error'));
			})
			.finally(() => {
				if (active) setLoading(false);
			});
		return (): void => {
			active = false;
		};
	}, [load, t]);

	const columns = useMemo<ColumnsType<RemediationExecution>>(
		() => [
			{
				title: t('history_col_time'),
				key: 'time',
				render: (_, r): string => r.terminalAt || r.proposedAt || '',
			},
			{ title: t('history_col_sop'), dataIndex: 'sopId', key: 'sopId' },
			{
				title: t('history_col_status'),
				key: 'status',
				render: (_, r): JSX.Element => <RemediationStatusBadge status={r.status} />,
			},
			{
				title: t('history_col_exit'),
				key: 'exit',
				render: (_, r): string =>
					typeof r.exitCode === 'number' ? String(r.exitCode) : '-',
			},
			{ title: t('history_col_approver'), dataIndex: 'approvedBy', key: 'approvedBy' },
		],
		[t],
	);

	return (
		<section className="sop-documents-page__section">
			<div className="sop-documents-page__binding">
				<Select
					style={{ minWidth: 160 }}
					value={statusFilter || 'all'}
					onChange={(v): void => setStatusFilter(v === 'all' ? '' : v)}
					options={[
						{ value: 'all', label: t('history_filter_status_all') },
						...STATUS_OPTIONS.map((s) => ({ value: s, label: s })),
					]}
				/>
				<Input
					placeholder={t('history_filter_sop_placeholder')}
					value={sopFilter}
					onChange={(e): void => setSopFilter(e.target.value)}
					allowClear
				/>
			</div>
			{error && (
				<Typography.Text type="danger">{error}</Typography.Text>
			)}
			<Table
				columns={columns}
				dataSource={rows}
				loading={loading}
				rowKey="id"
				size="small"
				locale={{ emptyText: t('history_empty') }}
				expandable={{
					expandedRowRender: (r): JSX.Element => <RemediationResult rem={r} />,
				}}
				pagination={{ hideOnSinglePage: true, pageSize: 20, showSizeChanger: false }}
			/>
		</section>
	);
}

export default RemediationHistory;
