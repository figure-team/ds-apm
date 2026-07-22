import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import {
	Alert,
	Button,
	Form,
	Input,
	InputNumber,
	Modal,
	Popconfirm,
	Select,
	Switch,
	Table,
	Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import getConfig from 'api/codeRca/getConfig';
import updateConfig from 'api/codeRca/updateConfig';
import listRepos from 'api/codeRca/listRepos';
import upsertRepo from 'api/codeRca/upsertRepo';
import deleteRepo from 'api/codeRca/deleteRepo';
import listServiceMaps from 'api/codeRca/listServiceMaps';
import upsertServiceMap from 'api/codeRca/upsertServiceMap';
import deleteServiceMap from 'api/codeRca/deleteServiceMap';
import {
	CodeRcaConfig,
	CodebaseRepo,
	CodebaseServiceMap,
	CREDENTIAL_UNCHANGED,
} from 'api/codeRca/types';

const CONTRACT_VERSION_REPO = 'ds.codebase_repo.v1';

interface Props {
	isAdmin: boolean;
}

function ConfigTab({ isAdmin }: Props): JSX.Element {
	const { t } = useTranslation(['codeRca']);

	// ── Config state ──────────────────────────────────────────────────────────
	const [config, setConfig] = useState<CodeRcaConfig | null>(null);
	const [isSaving, setIsSaving] = useState(false);

	// ── Repos state ───────────────────────────────────────────────────────────
	const [repos, setRepos] = useState<CodebaseRepo[]>([]);
	const [repoModalOpen, setRepoModalOpen] = useState(false);
	const [editingRepo, setEditingRepo] = useState<CodebaseRepo | null>(null);
	const [credentialTouched, setCredentialTouched] = useState(false);
	const [repoForm] = Form.useForm();

	// ── Service maps state ────────────────────────────────────────────────────
	const [serviceMaps, setServiceMaps] = useState<CodebaseServiceMap[]>([]);
	const [mapForm] = Form.useForm();

	// ── Load on mount ─────────────────────────────────────────────────────────
	useEffect(() => {
		let cancelled = false;

		const load = async (): Promise<void> => {
			try {
				const [cfgRes, reposRes, mapsRes] = await Promise.all([
					getConfig(),
					listRepos(),
					listServiceMaps(),
				]);
				if (cancelled) return;
				setConfig(cfgRes.data);
				setRepos(reposRes.data);
				setServiceMaps(mapsRes.data);
			} catch {
				// silently ignore load errors; individual save will surface errors
			}
		};

		void load();
		return (): void => {
			cancelled = true;
		};
	}, []);

	// ── Config save ───────────────────────────────────────────────────────────
	const handleSaveConfig = useCallback(async (): Promise<void> => {
		if (!config) return;
		setIsSaving(true);
		try {
			await updateConfig(config);
			toast.success(t('saved'));
		} catch {
			toast.error(t('save_failed'));
		} finally {
			setIsSaving(false);
		}
	}, [config, t]);

	// ── Repo modal ────────────────────────────────────────────────────────────
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
			if (err && typeof err === 'object' && 'errorFields' in err) return;
			toast.error(t('save_failed'));
		}
	}, [repoForm, editingRepo, t]);

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
		[t],
	);

	// ── Service map add ───────────────────────────────────────────────────────
	const handleAddMap = useCallback(async (): Promise<void> => {
		try {
			const values = await mapForm.validateFields();
			const payload: CodebaseServiceMap = {
				orgId: '',
				serviceName: values.serviceName,
				repoId: values.repoId,
				subpath: values.subpath ?? '',
			};
			await upsertServiceMap(payload);
			const mapsRes = await listServiceMaps();
			setServiceMaps(mapsRes.data);
			mapForm.resetFields();
			toast.success(t('saved'));
		} catch (err: unknown) {
			if (err && typeof err === 'object' && 'errorFields' in err) return;
			toast.error(t('save_failed'));
		}
	}, [mapForm, t]);

	const handleDeleteMap = useCallback(
		async (serviceName: string): Promise<void> => {
			try {
				await deleteServiceMap(serviceName);
				setServiceMaps((prev) =>
					prev.filter((m) => m.serviceName !== serviceName),
				);
				toast.success(t('saved'));
			} catch {
				toast.error(t('save_failed'));
			}
		},
		[t],
	);

	// ── Repo table columns ────────────────────────────────────────────────────
	const repoColumns: ColumnsType<CodebaseRepo> = [
		{ title: t('repo_id'), dataIndex: 'repoId', key: 'repoId' },
		{ title: t('repo_git_url'), dataIndex: 'gitUrl', key: 'gitUrl', ellipsis: true },
		{ title: t('repo_default_branch'), dataIndex: 'defaultBranch', key: 'defaultBranch' },
		{
			title: t('repo_enabled'),
			dataIndex: 'enabled',
			key: 'enabled',
			render: (val: boolean): string => (val ? '✓' : '✗'),
		},
		{ title: t('repo_last_sync'), dataIndex: 'lastSyncStatus', key: 'lastSyncStatus' },
		{
			title: t('repo_baseline'),
			dataIndex: 'baselineCommit',
			key: 'baselineCommit',
			render: (val: string): JSX.Element => <code>{val}</code>,
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

	// ── Service map table columns ─────────────────────────────────────────────
	const mapColumns: ColumnsType<CodebaseServiceMap> = [
		{ title: t('map_service'), dataIndex: 'serviceName', key: 'serviceName' },
		{ title: t('map_repo'), dataIndex: 'repoId', key: 'repoId' },
		{ title: t('map_subpath'), dataIndex: 'subpath', key: 'subpath' },
		{
			title: '',
			key: 'actions',
			render: (_: unknown, row: CodebaseServiceMap): JSX.Element => (
				<Popconfirm
					title={t('map_delete_confirm')}
					onConfirm={(): Promise<void> => handleDeleteMap(row.serviceName)}
					disabled={!isAdmin}
				>
					<Button size="small" danger disabled={!isAdmin}>
						{t('delete')}
					</Button>
				</Popconfirm>
			),
		},
	];

	return (
		<div>
			{/* Card 1: Feature + Thresholds */}
			<section className="code-rca-settings__card">
				<h3 className="code-rca-settings__card-title">{t('field_enabled')}</h3>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_enabled')}</label>
					<Switch
						checked={config?.enabled ?? false}
						onChange={(val): void =>
							setConfig((prev) => (prev ? { ...prev, enabled: val } : prev))
						}
						disabled={!isAdmin}
						style={{ alignSelf: 'flex-start' }}
					/>
				</div>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_min_severity')}</label>
					<Select
						value={config?.minSeverity ?? 'error'}
						onChange={(val): void =>
							setConfig((prev) => (prev ? { ...prev, minSeverity: val } : prev))
						}
						disabled={!isAdmin}
						style={{ width: 180 }}
						options={[
							{ value: 'critical', label: 'critical' },
							{ value: 'error', label: 'error' },
							{ value: 'warning', label: 'warning' },
							{ value: 'info', label: 'info' },
						]}
					/>
				</div>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_cooldown')}</label>
					<InputNumber
						value={config?.cooldownWindowSecs ?? 0}
						min={0}
						onChange={(val): void =>
							setConfig((prev) =>
								prev ? { ...prev, cooldownWindowSecs: val ?? 0 } : prev,
							)
						}
						disabled={!isAdmin}
					/>
				</div>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_max_runs_per_day')}</label>
					<InputNumber
						value={config?.maxRunsPerDay ?? 0}
						min={0}
						onChange={(val): void =>
							setConfig((prev) =>
								prev ? { ...prev, maxRunsPerDay: val ?? 0 } : prev,
							)
						}
						disabled={!isAdmin}
					/>
				</div>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_max_queue_depth')}</label>
					<InputNumber
						value={config?.maxQueueDepth ?? 0}
						min={0}
						onChange={(val): void =>
							setConfig((prev) =>
								prev ? { ...prev, maxQueueDepth: val ?? 0 } : prev,
							)
						}
						disabled={!isAdmin}
					/>
				</div>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_max_concurrent')}</label>
					<InputNumber
						value={config?.maxConcurrentRuns ?? 0}
						min={0}
						onChange={(val): void =>
							setConfig((prev) =>
								prev ? { ...prev, maxConcurrentRuns: val ?? 0 } : prev,
							)
						}
						disabled={!isAdmin}
					/>
				</div>

				<div className="code-rca-settings__field">
					<label className="code-rca-settings__field-label">{t('field_allow_unbound')}</label>
					<Switch
						checked={config?.allowUnboundWithoutAnomaly ?? false}
						onChange={(val): void =>
							setConfig((prev) =>
								prev ? { ...prev, allowUnboundWithoutAnomaly: val } : prev,
							)
						}
						disabled={!isAdmin}
						style={{ alignSelf: 'flex-start' }}
					/>
					{config?.allowUnboundWithoutAnomaly && (
						<Alert
							type="warning"
							showIcon
							message={t('allow_unbound_warning')}
							style={{ marginTop: 8 }}
						/>
					)}
				</div>

				<div className="code-rca-settings__actions">
					<Button
						type="primary"
						onClick={handleSaveConfig}
						loading={isSaving}
						disabled={!isAdmin}
					>
						{t('save')}
					</Button>
				</div>
			</section>

			{/* Card 2: Repos */}
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
			</section>

			{/* Card 3: Service maps */}
			<section className="code-rca-settings__card">
				<h3 className="code-rca-settings__card-title">{t('maps_title')}</h3>

				<Form form={mapForm} layout="inline" style={{ marginBottom: 12 }}>
					<Form.Item name="serviceName" rules={[{ required: true }]}>
						<Input placeholder={t('map_service')} disabled={!isAdmin} />
					</Form.Item>
					<Form.Item name="repoId" rules={[{ required: true }]}>
						<Select
							placeholder={t('map_repo')}
							disabled={!isAdmin}
							style={{ width: 200 }}
							showSearch
							options={repos.map((r) => ({ value: r.repoId, label: r.repoId }))}
						/>
					</Form.Item>
					<Form.Item name="subpath">
						<Input placeholder={t('map_subpath')} disabled={!isAdmin} />
					</Form.Item>
					<Form.Item>
						<Button onClick={handleAddMap} disabled={!isAdmin}>
							{t('map_add')}
						</Button>
					</Form.Item>
				</Form>

				<Table
					dataSource={serviceMaps}
					columns={mapColumns}
					rowKey="serviceName"
					size="small"
					pagination={false}
				/>
			</section>

			{/* Repo add/edit modal */}
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
					<Form.Item
						name="repoId"
						label={t('repo_id')}
						rules={[{ required: true }]}
					>
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
						tooltip={t('repo_artifact_path_hint')}
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
							placeholder={
								editingRepo ? t('credential_unchanged_hint') : undefined
							}
							onChange={(): void => {
								if (editingRepo) setCredentialTouched(true);
							}}
						/>
					</Form.Item>
					<Form.Item name="enabled" label={t('repo_enabled')} valuePropName="checked">
						<Switch defaultChecked />
					</Form.Item>
				</Form>
			</Modal>
		</div>
	);
}

export default ConfigTab;
