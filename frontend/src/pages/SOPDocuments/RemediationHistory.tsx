import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Input, Select, Table } from 'antd';
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
	const [statusFilter, setStatusFilter] = useState<string>('');
	const [sopFilter, setSopFilter] = useState<string>('');

	const load = useCallback(async (): Promise<void> => {
		setLoading(true);
		try {
			const data = await listRemediations({
				status: statusFilter || undefined,
				sopId: sopFilter || undefined,
				limit: LIST_LIMIT,
			});
			setRows(data);
		} finally {
			setLoading(false);
		}
	}, [statusFilter, sopFilter]);

	useEffect(() => {
		void load();
	}, [load]);

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
