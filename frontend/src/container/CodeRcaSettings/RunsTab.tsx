import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import { Alert, Button, Drawer, Input, Select, Spin, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import listRuns from 'api/codeRca/listRuns';
import getRun from 'api/codeRca/getRun';
import enqueueRun from 'api/codeRca/enqueueRun';
import exportRun from 'api/codeRca/exportRun';
import {
	CodeRcaRunDetail,
	CodeRcaRunStatus,
	CodeRcaRunSummary,
} from 'api/codeRca/types';
import { MarkdownRenderer } from 'components/MarkdownRenderer/MarkdownRenderer';
import { Bug, Info, Send, ShieldAlert, Wrench } from 'lucide-react';

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
	const [exporting, setExporting] = useState(false);
	const [testService, setTestService] = useState('');
	const [enqueuing, setEnqueuing] = useState(false);

	const fetchRuns = useCallback(async (): Promise<void> => {
		try {
			const res = await listRuns(statusFilter ? { status: statusFilter } : {});
			setRuns(res.data);
		} catch {
			toast.error(t('toast_runs_load_failed'));
		}
	}, [statusFilter, t]);

	// Poll a manually-enqueued run until it reaches a terminal state, refreshing
	// the list so the user watches it progress queued → running → done.
	const pollRun = useCallback(
		(runId: string): void => {
			let tries = 0;
			const iv = setInterval(async () => {
				tries += 1;
				try {
					const res = await getRun(runId);
					const st = res.data.status as string;
					void fetchRuns();
					if (['done', 'failed', 'timeout', 'unparseable'].includes(st)) {
						clearInterval(iv);
						if (st === 'done')
							toast.success(t('toast_run_done', { status: st }));
						else toast.error(t('toast_run_ended', { status: st }));
					}
				} catch {
					// keep polling
				}
				if (tries > 48) clearInterval(iv);
			}, 5000);
		},
		[fetchRuns, t],
	);

	const handleTestRun = useCallback(async (): Promise<void> => {
		const svc = testService.trim();
		if (!svc) {
			toast.error(t('toast_service_required'));
			return;
		}
		setEnqueuing(true);
		try {
			const res = await enqueueRun(svc);
			if (!res.admitted) {
				toast.error(
					t('toast_run_rejected', {
						reason: res.reason || t('toast_reason_limit'),
					}),
				);
				return;
			}
			toast.success(t('toast_run_enqueued'));
			void fetchRuns();
			pollRun(res.runId);
		} catch (e) {
			toast.error(
				(e as { response?: { data?: { error?: { message?: string } } } })?.response
					?.data?.error?.message ?? t('toast_run_failed'),
			);
		} finally {
			setEnqueuing(false);
		}
	}, [testService, fetchRuns, pollRun, t]);

	useEffect(() => {
		void fetchRuns();
	}, [fetchRuns]);

	const handleExport = useCallback(async (): Promise<void> => {
		if (!detail) return;
		setExporting(true);
		try {
			const res = await exportRun(detail.runId);
			// 경로 전체를 제목에 넣으면 줄바꿈되는데, @signozhq/ui 토스트 제목은
			// height 20px 고정이라 넘친 줄이 박스 밖으로 겹쳐 보인다. 경로는
			// 높이 제한이 없는 description으로 내린다.
			toast.success(t('toast_export_success'), {
				description: res.data.path,
			});
		} catch (e) {
			toast.error(
				(e as { response?: { data?: { error?: { message?: string } } } })?.response
					?.data?.error?.message ?? t('toast_export_failed'),
			);
		} finally {
			setExporting(false);
		}
	}, [detail, t]);

	const handleRowClick = useCallback(
		async (row: CodeRcaRunSummary): Promise<void> => {
			setDrawerOpen(true);
			setDetail(null);
			setLoadingDetail(true);
			try {
				const res = await getRun(row.runId);
				setDetail(res.data);
			} catch {
				toast.error(t('toast_run_detail_load_failed'));
			} finally {
				setLoadingDetail(false);
			}
		},
		[t],
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
		{
			title: t('run_failure_reason'),
			dataIndex: 'failureReason',
			key: 'failureReason',
			ellipsis: true,
			render: (val: string): JSX.Element =>
				val ? <span style={{ color: '#cf1322' }}>{val}</span> : <span>-</span>,
		},
	];

	return (
		<div>
			<div className="code-rca-settings__runs-toolbar">
				<Input
					value={testService}
					onChange={(e): void => setTestService(e.target.value)}
					placeholder="payment-api"
					style={{ width: 180 }}
					onPressEnter={handleTestRun}
				/>
				<Button type="primary" loading={enqueuing} onClick={handleTestRun}>
					테스트 실행
				</Button>
				<span style={{ flex: 1 }} />
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
				width={680}
				rootClassName="code-rca-drawer"
				title={
					<div className="code-rca-drawer__title">
						<span className="code-rca-drawer__title-svc">{detail?.service ?? ''}</span>
						{detail && (
							<Tag color={statusColor(detail.status)}>{detail.status}</Tag>
						)}
					</div>
				}
			>
				<Spin spinning={loadingDetail}>
					{detail && (
						<div className="code-rca-report">
							{detail.status === 'done' && (
								<Button
									type="primary"
									icon={<Send size={14} />}
									loading={exporting}
									onClick={handleExport}
									style={{ marginBottom: 12 }}
								>
									ds-navi에 산출물 전송
								</Button>
							)}

							{detail.status === 'done' && (
								<div className="code-rca-report__hitl">
									<ShieldAlert size={15} />
									<span>{t('run_hitl_notice')}</span>
								</div>
							)}

							{detail.failureReason && (
								<Alert
									type="error"
									showIcon
									message={t('run_failure_reason')}
									description={
										<pre className="code-rca-settings__report">
											{detail.failureReason}
										</pre>
									}
									style={{ marginBottom: 16 }}
								/>
							)}

							{detail.status === 'done' && (
								<>
									<section className="code-rca-card code-rca-card--cause">
										<header className="code-rca-card__head">
											<div className="code-rca-card__title">
												<Bug size={15} />
												<span>{t('run_root_cause')}</span>
											</div>
											{detail.confidence && (
												<span
													className={`code-rca-confidence code-rca-confidence--${detail.confidence}`}
												>
													<span className="code-rca-confidence__dot" />
													{t('run_confidence')}:{' '}
													{t(`run_confidence_${detail.confidence}`, detail.confidence)}
												</span>
											)}
										</header>
										<MarkdownRenderer
											className="code-rca-md"
											markdownContent={detail.rootCause}
											variables={{}}
										/>
									</section>

									<section className="code-rca-card code-rca-card--fix">
										<header className="code-rca-card__head">
											<div className="code-rca-card__title">
												<Wrench size={15} />
												<span>{t('run_proposed_fix')}</span>
											</div>
											{detail.baselineCommit && (
												<code className="code-rca-card__commit">
													{detail.baselineCommit.slice(0, 8)}
												</code>
											)}
										</header>
										<MarkdownRenderer
											className="code-rca-md"
											markdownContent={detail.proposedFix}
											variables={{}}
										/>
									</section>

									{detail.limitations && (
										<section className="code-rca-card code-rca-card--limits">
											<header className="code-rca-card__head">
												<div className="code-rca-card__title">
													<Info size={15} />
													<span>{t('run_limitations')}</span>
												</div>
											</header>
											<MarkdownRenderer
												className="code-rca-md"
												markdownContent={detail.limitations}
												variables={{}}
											/>
										</section>
									)}
								</>
							)}

							{detail.status !== 'done' && detail.baselineCommit && (
								<div className="code-rca-report__meta">
									<span className="code-rca-report__meta-label">
										{t('run_baseline')}
									</span>
									<code>{detail.baselineCommit}</code>
								</div>
							)}
						</div>
					)}
				</Spin>
			</Drawer>
		</div>
	);
}

export default RunsTab;
