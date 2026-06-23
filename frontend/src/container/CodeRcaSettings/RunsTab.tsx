import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import { Alert, Button, Drawer, Input, Select, Spin, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import listRuns from 'api/codeRca/listRuns';
import getRun from 'api/codeRca/getRun';
import enqueueRun from 'api/codeRca/enqueueRun';
import {
	CodeRcaRunDetail,
	CodeRcaRunStatus,
	CodeRcaRunSummary,
} from 'api/codeRca/types';
import { MarkdownRenderer } from 'components/MarkdownRenderer/MarkdownRenderer';
import { Bug, Info, ShieldAlert, Wrench } from 'lucide-react';

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
	const [testService, setTestService] = useState('');
	const [enqueuing, setEnqueuing] = useState(false);

	const fetchRuns = useCallback(async (): Promise<void> => {
		try {
			const res = await listRuns(statusFilter ? { status: statusFilter } : {});
			setRuns(res.data);
		} catch {
			toast.error('Failed to load runs');
		}
	}, [statusFilter]);

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
						if (st === 'done') toast.success(`코드 RCA 분석 완료 (${st})`);
						else toast.error(`코드 RCA 분석 종료: ${st}`);
					}
				} catch {
					// keep polling
				}
				if (tries > 48) clearInterval(iv);
			}, 5000);
		},
		[fetchRuns],
	);

	const handleTestRun = useCallback(async (): Promise<void> => {
		const svc = testService.trim();
		if (!svc) {
			toast.error('서비스 이름을 입력하세요 (예: payment-api)');
			return;
		}
		setEnqueuing(true);
		try {
			const res = await enqueueRun(svc);
			if (!res.admitted) {
				toast.error(`실행이 거부되었습니다: ${res.reason || '한도 초과'}`);
				return;
			}
			toast.success('테스트 실행을 큐에 등록했습니다. 분석 진행 중…');
			void fetchRuns();
			pollRun(res.runId);
		} catch (e) {
			toast.error(
				(e as { response?: { data?: { error?: { message?: string } } } })?.response
					?.data?.error?.message ?? '실행에 실패했습니다.',
			);
		} finally {
			setEnqueuing(false);
		}
	}, [testService, fetchRuns, pollRun]);

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
