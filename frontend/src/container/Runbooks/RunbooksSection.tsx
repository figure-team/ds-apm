import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Dropdown, Tabs, Empty, Spin, Modal } from 'antd';
import { DownloadOutlined, PlusOutlined } from '@ant-design/icons';
import { toast } from '@signozhq/ui';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';
import createRunbook from 'api/runbook/createRunbook';
import listRunbooks from 'api/runbook/listRunbooks';
import updateRunbook from 'api/runbook/updateRunbook';
import deleteRunbook from 'api/runbook/deleteRunbook';
import { Runbook, RunbookStatus } from './types';
import RunbookCard from './RunbookCard';
import RunbookForm from './RunbookForm';
import RunbookDraftFromError from './RunbookDraftFromError';
import RunbookBulkUploadModal from './RunbookBulkUploadModal';
import RunbookBulkPreviewDrawer from './RunbookBulkPreviewDrawer';
import {
	downloadRunbookExcelTemplate,
	type ParseRunbookExcelResult,
} from './parseRunbookExcel';
import './Runbooks.styles.scss';

interface Props {
	sopId: string;
	version: string;
}

type StatusFilter = 'approved,draft' | 'approved' | 'draft' | 'deprecated';

export default function RunbooksSection({ sopId, version }: Props): JSX.Element {
	const { t } = useTranslation(['runbooks']);
	const { user } = useAppContext();
	const [runbooks, setRunbooks] = useState<Runbook[]>([]);
	const [statusFilter, setStatusFilter] = useState<StatusFilter>('approved,draft');
	const [loading, setLoading] = useState(false);
	const [editing, setEditing] = useState<Runbook | null>(null);
	const [creatingNew, setCreatingNew] = useState(false);
	const [saving, setSaving] = useState(false);
	const [aiDraftOpen, setAiDraftOpen] = useState(false);
	const [bulkUploadOpen, setBulkUploadOpen] = useState(false);
	const [bulkPreviewOpen, setBulkPreviewOpen] = useState(false);
	const [bulkParseResult, setBulkParseResult] =
		useState<ParseRunbookExcelResult | null>(null);

	const canEdit =
		user.role === USER_ROLES.ADMIN || user.role === USER_ROLES.EDITOR;
	const canDelete = user.role === USER_ROLES.ADMIN;

	const fetchRunbooks = useCallback(async (filter: StatusFilter) => {
		setLoading(true);
		try {
			const response = await listRunbooks(sopId, version, filter);
			setRunbooks(response.data.runbooks);
		} catch (error) {
			toast.error(t('toast_load_error'));
			console.error(error);
		} finally {
			setLoading(false);
		}
	}, [sopId, version, t]);

	useEffect(() => {
		fetchRunbooks(statusFilter);
	}, [statusFilter, fetchRunbooks]);

	const handleStatusChange = useCallback(
		async (runbook: Runbook, nextStatus: RunbookStatus) => {
			if (loading) return; // TODO: replace with proper request-id versioning post-v0.1
			try {
				await updateRunbook(sopId, version, {
					...runbook,
					status: nextStatus,
				});
				await fetchRunbooks(statusFilter);
				toast.success(
					nextStatus === 'deprecated'
						? t('toast_status_deprecated')
						: t('toast_status_approved')
				);
			} catch (error) {
				toast.error(t('toast_status_error'));
				console.error(error);
			}
		},
		[loading, sopId, version, statusFilter, fetchRunbooks, t]
	);

	const handleDelete = useCallback(
		async (runbook: Runbook) => {
			if (loading) return; // TODO: replace with proper request-id versioning post-v0.1
			try {
				await deleteRunbook(sopId, version, runbook.id);
				await fetchRunbooks(statusFilter);
				toast.success(t('toast_deleted'));
			} catch (error) {
				toast.error(t('toast_delete_error'));
				console.error(error);
			}
		},
		[loading, sopId, version, statusFilter, fetchRunbooks, t]
	);

	const handleEdit = useCallback((runbook: Runbook) => {
		setEditing(runbook);
	}, []);

	const handleNewRunbook = useCallback(() => {
		setCreatingNew(true);
	}, []);

	const handleBulkParsed = useCallback((result: ParseRunbookExcelResult) => {
		setBulkParseResult(result);
		setBulkUploadOpen(false);
		setBulkPreviewOpen(true);
	}, []);

	const handleBulkRegistered = useCallback(() => {
		void fetchRunbooks(statusFilter);
	}, [fetchRunbooks, statusFilter]);

	const handleFormCancel = useCallback(() => {
		setEditing(null);
		setCreatingNew(false);
	}, []);

	const handleFormSubmit = useCallback(
		async (values: Partial<Runbook>) => {
			setSaving(true);
			try {
				if (editing) {
					await updateRunbook(sopId, version, { ...editing, ...values });
					toast.success(t('toast_updated'));
				} else {
					await createRunbook(sopId, version, values);
					toast.success(t('toast_created'));
				}
				setEditing(null);
				setCreatingNew(false);
				await fetchRunbooks(statusFilter);
			} catch (error) {
				toast.error(t('toast_save_error'));
				console.error(error);
			} finally {
				setSaving(false);
			}
		},
		[editing, sopId, version, statusFilter, fetchRunbooks, t]
	);

	const showForm = editing !== null || creatingNew;

	const stats = useMemo(() => {
		const approved = runbooks.filter((rb) => rb.status === 'approved').length;
		const draft = runbooks.filter((rb) => rb.status === 'draft').length;
		const deprecated = runbooks.filter((rb) => rb.status === 'deprecated').length;
		return {
			active: approved + draft,
			approved,
			draft,
			deprecated,
		};
	}, [runbooks]);

	const tabItems = [
		{
			key: 'approved,draft',
			label: t('tab_active', { count: stats.active }),
		},
		{
			key: 'approved',
			label: t('tab_approved', { count: stats.approved }),
		},
		{
			key: 'draft',
			label: t('tab_drafts', { count: stats.draft }),
		},
		{
			key: 'deprecated',
			label: t('tab_deprecated', { count: stats.deprecated }),
		},
	];

	return (
		<div className="runbooks-section">
			<div className="runbooks-section__header">
				<h2>{t('section_title')}</h2>
				{canEdit && (
					<div className="runbooks-section__header-actions">
						<Button
							icon={<DownloadOutlined />}
							onClick={downloadRunbookExcelTemplate}
						>
							{t('btn_template')}
						</Button>
						<Dropdown
							menu={{
								items: [
									{ key: 'manual', label: t('menu_manual'), onClick: handleNewRunbook },
									{
										key: 'ai',
										label: t('menu_ai_draft'),
										onClick: (): void => setAiDraftOpen(true),
									},
									{ type: 'divider' },
									{
										key: 'bulk',
										label: t('menu_bulk_upload'),
										onClick: (): void => setBulkUploadOpen(true),
									},
								],
							}}
						>
							<Button type="primary" icon={<PlusOutlined />}>
								{t('btn_new')}
							</Button>
						</Dropdown>
					</div>
				)}
			</div>

			<Tabs
				activeKey={statusFilter}
				items={tabItems}
				onChange={(key) => setStatusFilter(key as StatusFilter)}
			/>

			<Spin spinning={loading}>
				{!loading && runbooks.length === 0 && (
					<Empty description={t('empty')} />
				)}
				{runbooks.length > 0 && (
					<div className="runbooks-section__list">
						{runbooks.map((runbook) => (
							<RunbookCard
								key={runbook.id}
								runbook={runbook}
								canEdit={canEdit}
								canDelete={canDelete}
								onEdit={handleEdit}
								onStatusChange={handleStatusChange}
								onDelete={handleDelete}
							/>
						))}
					</div>
				)}
			</Spin>

			<Modal
				open={showForm}
				title={editing ? t('modal_edit_title') : t('modal_new_title')}
				onCancel={handleFormCancel}
				footer={null}
				width={720}
				destroyOnClose
			>
				{showForm && (
					<RunbookForm
						initial={editing ?? undefined}
						onSubmit={handleFormSubmit}
						onCancel={handleFormCancel}
						saving={saving}
					/>
				)}
			</Modal>

			<RunbookDraftFromError
				sopId={sopId}
				version={version}
				open={aiDraftOpen}
				onSaved={(): void => {
					setAiDraftOpen(false);
					void fetchRunbooks(statusFilter);
				}}
				onCancel={(): void => setAiDraftOpen(false)}
			/>

			<RunbookBulkUploadModal
				open={bulkUploadOpen}
				onClose={(): void => setBulkUploadOpen(false)}
				onParsed={handleBulkParsed}
			/>

			<RunbookBulkPreviewDrawer
				open={bulkPreviewOpen}
				parseResult={bulkParseResult}
				sopId={sopId}
				version={version}
				onClose={(): void => setBulkPreviewOpen(false)}
				onRegistered={handleBulkRegistered}
			/>
		</div>
	);
}
