import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Alert, Badge, Button, Empty, Modal, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
	deleteRemediationTarget,
	listRemediationTargets,
	RemediationTargetWire,
	testRemediationConnection,
} from 'api/remediationTargets';

import TargetFormDrawer from './TargetFormDrawer';

import './RemediationTargetSettings.styles.scss';

// 마지막 연결 테스트 결과 — 세션 로컬 상태(서버 저장 안 함, 스펙 §4.2).
type RowTestState =
	| { status: 'success' }
	| { status: 'fail'; error?: string };

const FINGERPRINT_PREVIEW_LEN = 16;

function RemediationTargetSettings(): JSX.Element {
	const { t } = useTranslation(['routes']);
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

	const load = useCallback(async (): Promise<void> => {
		setLoading(true);
		try {
			const res = await listRemediationTargets();
			setTargets(res.targets ?? []);
			setEncryptionReady(res.encryptionReady);
		} catch {
			// 로드 실패는 조용히 무시 — 개별 작업에서 에러가 드러난다.
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		void load();
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
				title: '자동대응 타겟 삭제',
				content: `'${row.name}' 타겟을 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.`,
				icon: (
					<ExclamationCircleOutlined
						style={{ color: 'var(--danger-background)' }}
					/>
				),
				okText: '삭제',
				okButtonProps: { danger: true },
				cancelText: '취소',
				centered: true,
				onOk: async (): Promise<void> => {
					await deleteRemediationTarget(row.id);
					await load();
				},
			});
		},
		[modal, load],
	);

	const columns: ColumnsType<RemediationTargetWire> = [
		{
			title: '이름',
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
			title: '서비스 셀렉터',
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
			title: '지문',
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
			title: '마지막 테스트',
			key: 'lastTest',
			render: (_: unknown, row: RemediationTargetWire): JSX.Element => {
				const result = testResults[row.id];
				if (!result) {
					return <Badge status="default" text="미실행" />;
				}
				if (result.status === 'success') {
					return <Badge status="success" text="성공" />;
				}
				return (
					<Tooltip title={result.error}>
						<Badge status="error" text="실패" />
					</Tooltip>
				);
			},
		},
		{
			title: '작업',
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
						테스트
					</Button>
					<Button size="small" onClick={(): void => openEdit(row)}>
						수정
					</Button>
					<Button
						size="small"
						danger
						onClick={(): void => confirmDelete(row)}
					>
						삭제
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
						SSH 원격 자동대응을 실행할 대상 서버를 등록하고 연결을 테스트합니다.
					</p>
				</div>
				<Button
					type="primary"
					onClick={openCreate}
					disabled={!encryptionReady}
				>
					타겟 추가
				</Button>
			</header>

			{!encryptionReady && (
				<Alert
					type="warning"
					showIcon
					className="remediation-target-settings__banner"
					message="암호화 마스터키가 설정되지 않아 원격 타겟을 등록할 수 없습니다 (DS_APM_AI_CONFIG_ENCRYPTION_KEY)"
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
							description="등록된 타겟이 없습니다"
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
