import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Alert, Badge, Button, Empty, Modal, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { TFunction } from 'i18next';
import {
	deleteRemediationTarget,
	listRemediationTargets,
	RemediationTargetWire,
	TargetHealthWire,
	testRemediationConnection,
} from 'api/remediationTargets';

import TargetFormDrawer from './TargetFormDrawer';

import './RemediationTargetSettings.styles.scss';

// 마지막 연결 테스트 결과 — 세션 로컬 상태(서버 저장 안 함, 스펙 §4.2).
type RowTestState =
	| { status: 'success' }
	| { status: 'fail'; error?: string };

const FINGERPRINT_PREVIEW_LEN = 16;
const REFRESH_INTERVAL_MS = 60_000;

// 헬스 배지 매핑 — unknown/부재는 매핑에 없고 '확인 중' 기본 분기로 처리.
const HEALTH_BADGE: Partial<
	Record<
		TargetHealthWire['status'],
		{ badge: 'success' | 'error' | 'warning'; textKey: string }
	>
> = {
	healthy: { badge: 'success', textKey: 'health_healthy' },
	unreachable: { badge: 'error', textKey: 'health_unreachable' },
	mismatch: { badge: 'warning', textKey: 'health_mismatch' },
};

function renderHealthBadge(
	t: TFunction,
	health?: TargetHealthWire,
): JSX.Element {
	const meta = health && HEALTH_BADGE[health.status];
	if (!health || !meta) {
		return <Badge status="default" text={t('health_checking').toString()} />;
	}
	const lines: string[] = [];
	if (health.status === 'mismatch') {
		lines.push(t('health_mismatch_hint').toString());
	} else if (health.error) {
		lines.push(health.error);
	}
	if (health.checkedAt) {
		lines.push(
			t('health_last_checked', {
				time: new Date(health.checkedAt).toLocaleString(),
			}).toString(),
		);
	}
	const badge = <Badge status={meta.badge} text={t(meta.textKey).toString()} />;
	if (lines.length === 0) {
		return badge;
	}
	return (
		<Tooltip title={lines.join('\n')}>
			<span>{badge}</span>
		</Tooltip>
	);
}

function RemediationTargetSettings(): JSX.Element {
	const { t } = useTranslation(['remediation_targets', 'routes']);
	const [modal, contextHolder] = Modal.useModal();

	const [targets, setTargets] = useState<RemediationTargetWire[]>([]);
	const [encryptionReady, setEncryptionReady] = useState(true);
	const [loading, setLoading] = useState(false);

	// 행별 테스트 결과/진행 상태 (targetId 기준)
	const [testResults, setTestResults] = useState<Record<string, RowTestState>>(
		{},
	);
	const [testingId, setTestingId] = useState<string | null>(null);

	// Drawer
	const [drawerOpen, setDrawerOpen] = useState(false);
	const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
	const [editingTarget, setEditingTarget] = useState<
		RemediationTargetWire | undefined
	>(undefined);

	const load = useCallback(async (silent = false): Promise<void> => {
		if (!silent) {
			setLoading(true);
		}
		try {
			const res = await listRemediationTargets();
			setTargets(res.targets ?? []);
			setEncryptionReady(res.encryptionReady);
		} catch {
			// 로드 실패는 조용히 무시 — 개별 작업에서 에러가 드러난다.
		} finally {
			if (!silent) {
				setLoading(false);
			}
		}
	}, []);

	useEffect(() => {
		void load();
		// 백엔드 헬스체커(기본 300초 순회 + 저장 직후 Poke) 결과를 주기 반영.
		// silent — 매 갱신마다 테이블 스피너가 번쩍이지 않게 한다 (스펙 §4.3).
		const timer = setInterval(() => {
			void load(true);
		}, REFRESH_INTERVAL_MS);
		return (): void => clearInterval(timer);
	}, [load]);

	const openCreate = useCallback((): void => {
		setDrawerMode('create');
		setEditingTarget(undefined);
		setDrawerOpen(true);
	}, []);

	const openEdit = useCallback((row: RemediationTargetWire): void => {
		setDrawerMode('edit');
		setEditingTarget(row);
		setDrawerOpen(true);
	}, []);

	const handleDrawerClose = useCallback((): void => {
		setDrawerOpen(false);
	}, []);

	const handleSaved = useCallback((): void => {
		setDrawerOpen(false);
		void load();
	}, [load]);

	const runRowTest = useCallback(
		async (row: RemediationTargetWire): Promise<void> => {
			setTestingId(row.id);
			try {
				const result = await testRemediationConnection({ targetId: row.id });
				setTestResults((prev) => ({
					...prev,
					[row.id]: result.ok
						? { status: 'success' }
						: { status: 'fail', error: result.error },
				}));
			} catch {
				setTestResults((prev) => ({
					...prev,
					[row.id]: { status: 'fail' },
				}));
			} finally {
				setTestingId(null);
			}
		},
		[],
	);

	const confirmDelete = useCallback(
		(row: RemediationTargetWire): void => {
			modal.confirm({
				title: t('delete_modal_title'),
				content: t('delete_modal_content', { name: row.name }),
				icon: (
					<ExclamationCircleOutlined
						style={{ color: 'var(--danger-background)' }}
					/>
				),
				okText: t('btn_delete'),
				okButtonProps: { danger: true },
				cancelText: t('btn_cancel'),
				centered: true,
				onOk: async (): Promise<void> => {
					await deleteRemediationTarget(row.id);
					await load();
				},
			});
		},
		[modal, load, t],
	);

	const columns: ColumnsType<RemediationTargetWire> = [
		{
			title: t('col_name'),
			dataIndex: 'name',
			key: 'name',
		},
		{
			title: 'Host:Port',
			key: 'hostPort',
			render: (_: unknown, row: RemediationTargetWire): string =>
				`${row.host}:${row.port}`,
		},
		{
			title: 'User',
			dataIndex: 'user',
			key: 'user',
		},
		{
			title: t('col_service_selectors'),
			key: 'serviceSelectors',
			render: (_: unknown, row: RemediationTargetWire): JSX.Element => (
				<>
					{(row.serviceSelectors ?? []).map((s) => (
						<Tag key={s}>{s}</Tag>
					))}
				</>
			),
		},
		{
			title: t('col_fingerprint'),
			key: 'fingerprint',
			render: (_: unknown, row: RemediationTargetWire): JSX.Element => {
				const fp = row.hostKeyFingerprint ?? '';
				const preview =
					fp.length > FINGERPRINT_PREVIEW_LEN
						? `${fp.slice(0, FINGERPRINT_PREVIEW_LEN)}…`
						: fp;
				return (
					<Tooltip title={fp}>
						<code>{preview}</code>
					</Tooltip>
				);
			},
		},
		{
			title: t('col_status'),
			key: 'health',
			render: (_: unknown, row: RemediationTargetWire): JSX.Element =>
				renderHealthBadge(t, row.health),
		},
		{
			title: t('col_last_test'),
			key: 'lastTest',
			render: (_: unknown, row: RemediationTargetWire): JSX.Element => {
				const result = testResults[row.id];
				if (!result) {
					return <Badge status="default" text={t('test_not_run')} />;
				}
				if (result.status === 'success') {
					return <Badge status="success" text={t('test_success')} />;
				}
				return (
					<Tooltip title={result.error}>
						<Badge status="error" text={t('test_fail')} />
					</Tooltip>
				);
			},
		},
		{
			title: t('col_actions'),
			key: 'actions',
			render: (_: unknown, row: RemediationTargetWire): JSX.Element => (
				<div className="remediation-target-settings__row-actions">
					<Button
						size="small"
						loading={testingId === row.id}
						onClick={(): void => {
							void runRowTest(row);
						}}
					>
						{t('btn_row_test')}
					</Button>
					<Button size="small" onClick={(): void => openEdit(row)}>
						{t('btn_edit')}
					</Button>
					<Button
						size="small"
						danger
						onClick={(): void => confirmDelete(row)}
					>
						{t('btn_delete')}
					</Button>
				</div>
			),
		},
	];

	return (
		<div className="settings-shell remediation-target-settings">
			<header className="remediation-target-settings__header">
				<div className="remediation-target-settings__header-text">
					<h1 className="remediation-target-settings__title">
						{t('routes:remediation_targets')}
					</h1>
					<p className="remediation-target-settings__subtitle">
						{t('page_subtitle')}
					</p>
				</div>
				<Button
					type="primary"
					onClick={openCreate}
					disabled={!encryptionReady}
				>
					{t('btn_add_target')}
				</Button>
			</header>

			{!encryptionReady && (
				<Alert
					type="warning"
					showIcon
					className="remediation-target-settings__banner"
					message={t('banner_no_master_key')}
				/>
			)}

			<Table
				className="remediation-target-settings__table"
				dataSource={targets}
				columns={columns}
				rowKey="id"
				size="small"
				loading={loading}
				pagination={false}
				locale={{
					emptyText: (
						<Empty
							image={Empty.PRESENTED_IMAGE_SIMPLE}
							description={t('empty_no_targets')}
						/>
					),
				}}
			/>

			<TargetFormDrawer
				open={drawerOpen}
				mode={drawerMode}
				initial={editingTarget}
				encryptionReady={encryptionReady}
				onClose={handleDrawerClose}
				onSaved={handleSaved}
			/>

			{contextHolder}
		</div>
	);
}

export default RemediationTargetSettings;
