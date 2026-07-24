import {
	type Dispatch,
	type SetStateAction,
	useCallback,
	useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import {
	Button,
	Form,
	Input,
	Modal,
	Popconfirm,
	Switch,
	Table,
	Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import deleteRepo from 'api/codeRca/deleteRepo';
import listRepos from 'api/codeRca/listRepos';
import upsertRepo from 'api/codeRca/upsertRepo';
import { CodebaseRepo, CREDENTIAL_UNCHANGED } from 'api/codeRca/types';

const CONTRACT_VERSION_REPO = 'ds.codebase_repo.v1';

type Props = {
	repos: CodebaseRepo[];
	setRepos: Dispatch<SetStateAction<CodebaseRepo[]>>;
	isAdmin: boolean;
};

/** 코드 저장소 목록과 추가/수정 모달을 다루는 카드. */
function RcaReposCard({ repos, setRepos, isAdmin }: Props): JSX.Element {
	const { t } = useTranslation(['codeRca']);
	const [repoModalOpen, setRepoModalOpen] = useState(false);
	const [editingRepo, setEditingRepo] = useState<CodebaseRepo | null>(null);
	const [credentialTouched, setCredentialTouched] = useState(false);
	const [repoForm] = Form.useForm();

	const openAddRepo = useCallback((): void => {
		setEditingRepo(null);
		setCredentialTouched(false);
		repoForm.resetFields();
		setRepoModalOpen(true);
	}, [repoForm]);

	const openEditRepo = useCallback(
		(row: CodebaseRepo): void => {
			setEditingRepo(row);
			setCredentialTouched(false);
			repoForm.setFieldsValue({
				repoId: row.repoId,
				gitUrl: row.gitUrl,
				defaultBranch: row.defaultBranch,
				credential: '',
				enabled: row.enabled,
				artifactPath: row.artifactPath,
			});
			setRepoModalOpen(true);
		},
		[repoForm],
	);

	const handleRepoModalOk = useCallback(async (): Promise<void> => {
		try {
			const values = await repoForm.validateFields();
			const isEdit = editingRepo !== null;
			const credential =
				isEdit && values.credential === ''
					? CREDENTIAL_UNCHANGED
					: values.credential;

			const payload: CodebaseRepo = {
				contractVersion: CONTRACT_VERSION_REPO,
				orgId: editingRepo?.orgId ?? '',
				repoId: values.repoId,
				gitUrl: values.gitUrl,
				defaultBranch: values.defaultBranch,
				credential,
				enabled: values.enabled ?? true,
				artifactPath: (values.artifactPath ?? '').trim(),
				branchName: editingRepo?.branchName ?? '',
				fetched: editingRepo?.fetched ?? false,
				baselineCommit: editingRepo?.baselineCommit ?? '',
				lastSyncAt: editingRepo?.lastSyncAt ?? '',
				lastSyncStatus: editingRepo?.lastSyncStatus ?? '',
			};

			await upsertRepo(payload);
			const reposRes = await listRepos();
			setRepos(reposRes.data);
			setRepoModalOpen(false);
			repoForm.resetFields();
			toast.success(t('saved'));
		} catch (err: unknown) {
			if (err && typeof err === 'object' && 'errorFields' in err) {
				return;
			}
			toast.error(t('save_failed'));
		}
	}, [repoForm, editingRepo, setRepos, t]);

	const handleDeleteRepo = useCallback(
		async (repoId: string): Promise<void> => {
			try {
				await deleteRepo(repoId);
				setRepos((prev) => prev.filter((r) => r.repoId !== repoId));
				toast.success(t('saved'));
			} catch {
				toast.error(t('save_failed'));
			}
		},
		[setRepos, t],
	);

	const repoColumns: ColumnsType<CodebaseRepo> = [
		{ title: t('repo_id'), dataIndex: 'repoId', key: 'repoId' },
		{
			title: t('repo_git_url'),
			dataIndex: 'gitUrl',
			key: 'gitUrl',
			ellipsis: true,
		},
		{
			title: t('repo_default_branch'),
			dataIndex: 'defaultBranch',
			key: 'defaultBranch',
		},
		{
			title: t('repo_enabled'),
			dataIndex: 'enabled',
			key: 'enabled',
			render: (val: boolean): string => (val ? '✓' : '✗'),
		},
		{
			title: t('repo_last_sync'),
			dataIndex: 'lastSyncStatus',
			key: 'lastSyncStatus',
		},
		{
			title: t('repo_baseline'),
			dataIndex: 'baselineCommit',
			key: 'baselineCommit',
			render: (val: string): JSX.Element => <span>{val}</span>,
		},
		{
			title: '',
			key: 'actions',
			render: (_: unknown, row: CodebaseRepo): JSX.Element => (
				<>
					<Button
						size="small"
						onClick={(): void => openEditRepo(row)}
						disabled={!isAdmin}
						style={{ marginRight: 8 }}
					>
						{t('edit')}
					</Button>
					<Popconfirm
						title={t('repo_delete_confirm')}
						onConfirm={(): Promise<void> => handleDeleteRepo(row.repoId)}
						disabled={!isAdmin}
					>
						<Button size="small" danger disabled={!isAdmin}>
							{t('delete')}
						</Button>
					</Popconfirm>
				</>
			),
		},
	];

	return (
		<section className="code-rca-settings__card">
			<h3 className="code-rca-settings__card-title">{t('repos_title')}</h3>
			<div style={{ marginBottom: 12 }}>
				<Button onClick={openAddRepo} disabled={!isAdmin}>
					{t('repo_add')}
				</Button>
			</div>
			<Table
				dataSource={repos}
				columns={repoColumns}
				rowKey="repoId"
				size="small"
				pagination={false}
			/>

			{/* antd Modal은 document.body로 포털되므로 JSX 위치는 렌더 결과에 영향이 없다. */}
			<Modal
				open={repoModalOpen}
				title={editingRepo ? t('repo_id') : t('repo_add')}
				onOk={handleRepoModalOk}
				onCancel={(): void => {
					setRepoModalOpen(false);
					repoForm.resetFields();
				}}
				destroyOnClose
			>
				<Form form={repoForm} layout="vertical">
					<Form.Item name="repoId" label={t('repo_id')} rules={[{ required: true }]}>
						<Input disabled={editingRepo !== null} />
					</Form.Item>
					<Form.Item
						name="gitUrl"
						label={t('repo_git_url')}
						rules={[{ required: true }]}
					>
						<Input />
					</Form.Item>
					<Form.Item
						name="defaultBranch"
						label={t('repo_default_branch')}
						rules={[{ required: true }]}
					>
						<Input />
					</Form.Item>
					<Form.Item
						name="artifactPath"
						label={t('repo_artifact_path')}
						extra={
							<span style={{ wordBreak: 'keep-all', overflowWrap: 'break-word' }}>
								{t('repo_artifact_path_help')}
							</span>
						}
					>
						<Input placeholder="/srv/m-project" />
					</Form.Item>
					<Form.Item
						name="credential"
						label={
							<span>
								{t('repo_credential')}
								{editingRepo && !credentialTouched && (
									<Tag color="success" style={{ marginLeft: 8 }}>
										{t('credential_saved')}
									</Tag>
								)}
							</span>
						}
					>
						<Input.Password
							placeholder={editingRepo ? t('credential_unchanged_hint') : undefined}
							onChange={(): void => {
								if (editingRepo) {
									setCredentialTouched(true);
								}
							}}
						/>
					</Form.Item>
					<Form.Item
						name="enabled"
						label={t('repo_enabled')}
						valuePropName="checked"
					>
						<Switch defaultChecked />
					</Form.Item>
				</Form>
			</Modal>
		</section>
	);
}

export default RcaReposCard;
