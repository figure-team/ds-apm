import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import { Alert, Button, Drawer, Select, Spin, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import listRuns from 'api/codeRca/listRuns';
import getRun from 'api/codeRca/getRun';
import {
	CodeRcaRunDetail,
	CodeRcaRunStatus,
	CodeRcaRunSummary,
} from 'api/codeRca/types';

const STATUS_OPTIONS = [
	{ value: '', label: 'All' },
	{ value: 'queued', label: 'queued' },
	{ value: 'running', label: 'running' },
	{ value: 'done', label: 'done' },
	{ value: 'failed', label: 'failed' },
	{ value: 'timeout', label: 'timeout' },
	{ value: 'unparseable', label: 'unparseable' },
];

function statusColor(status: CodeRcaRunStatus): string {
	switch (status) {
		case 'done':
			return 'green';
		case 'failed':
		case 'timeout':
			return 'red';
		case 'running':
			return 'blue';
		case 'unparseable':
			return 'orange';
		case 'queued':
		default:
			return 'default';
	}
}

function formatUnixSeconds(ts: number): string {
	if (!ts) return '';
	return new Date(ts * 1000).toLocaleString();
}

function RunsTab(): JSX.Element {
	const { t } = useTranslation(['codeRca']);

	const [statusFilter, setStatusFilter] = useState<string>('');
	const [runs, setRuns] = useState<CodeRcaRunSummary[]>([]);
	const [drawerOpen, setDrawerOpen] = useState(false);
	const [detail, setDetail] = useState<CodeRcaRunDetail | null>(null);
	const [loadingDetail, setLoadingDetail] = useState(false);

	const fetchRuns = useCallback(async (): Promise<void> => {
		try {
			const res = await listRuns(statusFilter ? { status: statusFilter } : {});
			setRuns(res.data);
		} catch {
			toast.error('Failed to load runs');
		}
	}, [statusFilter]);

	useEffect(() => {
		void fetchRuns();
	}, [fetchRuns]);

	const handleRowClick = useCallback(
		async (row: CodeRcaRunSummary): Promise<void> => {
			setDrawerOpen(true);
			setDetail(null);
			setLoadingDetail(true);
			try {
				const res = await getRun(row.runId);
				setDetail(res.data);
			} catch {
				toast.error('Failed to load run detail');
			} finally {
				setLoadingDetail(false);
			}
		},
		[],
	);

	const columns: ColumnsType<CodeRcaRunSummary> = [
		{
			title: t('run_created'),
			dataIndex: 'createdAt',
			key: 'createdAt',
			render: (val: number): string => formatUnixSeconds(val),
		},
		{
			title: t('run_service'),
			dataIndex: 'service',
			key: 'service',
		},
		{
			title: t('run_status'),
			dataIndex: 'status',
			key: 'status',
			render: (val: CodeRcaRunStatus): JSX.Element => (
				<Tag color={statusColor(val)}>{val}</Tag>
			),
		},
		{
			title: t('run_baseline'),
			dataIndex: 'baselineCommit',
			key: 'baselineCommit',
			render: (val: string): JSX.Element => <code>{val?.slice(0, 8)}</code>,
		},
		{
			title: t('run_attempts'),
			dataIndex: 'attempts',
			key: 'attempts',
		},
	];

	return (
		<div>
			<div className="code-rca-settings__runs-toolbar">
				<label>{t('runs_filter_status')}</label>
				<Select
					value={statusFilter}
					onChange={setStatusFilter}
					options={STATUS_OPTIONS}
					style={{ width: 160 }}
				/>
				<Button onClick={fetchRuns}>{t('runs_refresh')}</Button>
			</div>

			<Table
				dataSource={runs}
				columns={columns}
				rowKey="runId"
				size="small"
				locale={{ emptyText: t('run_empty') }}
				onRow={(row): { onClick: () => void } => ({
					onClick: (): void => {
						void handleRowClick(row);
					},
				})}
				style={{ cursor: 'pointer' }}
			/>

			<Drawer
				open={drawerOpen}
				onClose={(): void => setDrawerOpen(false)}
				width={600}
				title={detail?.service ?? ''}
			>
				<Spin spinning={loadingDetail}>
				{detail && (
					<div>
						<h3>{t('run_root_cause')}</h3>
						<pre className="code-rca-settings__report">{detail.rootCause}</pre>

						<h3>{t('run_proposed_fix')}</h3>
						<pre className="code-rca-settings__report">{detail.proposedFix}</pre>

						<div style={{ marginTop: 12 }}>
							<strong>{t('run_confidence')}: </strong>
							<Tag>{detail.confidence}</Tag>
						</div>

						{detail.limitations && (
							<div style={{ marginTop: 12 }}>
								<strong>{t('run_limitations')}: </strong>
								<span>{detail.limitations}</span>
							</div>
						)}

						<div style={{ marginTop: 12 }}>
							<strong>{t('run_baseline')}: </strong>
							<code>{detail.baselineCommit}</code>
						</div>

						<Alert
							type="warning"
							showIcon
							message={t('run_hitl_notice')}
							style={{ marginTop: 16 }}
						/>
					</div>
				)}
				</Spin>
			</Drawer>
		</div>
	);
}

export default RunsTab;
