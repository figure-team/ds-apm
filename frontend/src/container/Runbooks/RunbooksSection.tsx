import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Dropdown, Tabs, Empty, Spin, Modal } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
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
import './Runbooks.styles.scss';

interface Props {
	sopId: string;
	version: string;
}

type StatusFilter = 'approved,draft' | 'approved' | 'draft' | 'deprecated';

export default function RunbooksSection({ sopId, version }: Props): JSX.Element {
	const { user } = useAppContext();
	const [runbooks, setRunbooks] = useState<Runbook[]>([]);
	const [statusFilter, setStatusFilter] = useState<StatusFilter>('approved,draft');
	const [loading, setLoading] = useState(false);
	const [editing, setEditing] = useState<Runbook | null>(null);
	const [creatingNew, setCreatingNew] = useState(false);
	const [saving, setSaving] = useState(false);
	const [aiDraftOpen, setAiDraftOpen] = useState(false);

	const canEdit =
		user.role === USER_ROLES.ADMIN || user.role === USER_ROLES.EDITOR;
	const canDelete = user.role === USER_ROLES.ADMIN;

	const fetchRunbooks = useCallback(async (filter: StatusFilter) => {
		setLoading(true);
		try {
			const response = await listRunbooks(sopId, version, filter);
			setRunbooks(response.data.runbooks);
		} catch (error) {
			toast.error('Failed to load runbooks');
			console.error(error);
		} finally {
			setLoading(false);
		}
	}, [sopId, version]);

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
					`Runbook ${nextStatus === 'deprecated' ? 'deprecated' : 'approved'} successfully`
				);
			} catch (error) {
				toast.error('Failed to update runbook status');
				console.error(error);
			}
		},
		[loading, sopId, version, statusFilter, fetchRunbooks]
	);

	const handleDelete = useCallback(
		async (runbook: Runbook) => {
			if (loading) return; // TODO: replace with proper request-id versioning post-v0.1
			try {
				await deleteRunbook(sopId, version, runbook.id);
				await fetchRunbooks(statusFilter);
				toast.success('Runbook deleted successfully');
			} catch (error) {
				toast.error('Failed to delete runbook');
				console.error(error);
			}
		},
		[loading, sopId, version, statusFilter, fetchRunbooks]
	);

	const handleEdit = useCallback((runbook: Runbook) => {
		setEditing(runbook);
	}, []);

	const handleNewRunbook = useCallback(() => {
		setCreatingNew(true);
	}, []);

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
					toast.success('Runbook updated');
				} else {
					await createRunbook(sopId, version, values);
					toast.success('Runbook created');
				}
				setEditing(null);
				setCreatingNew(false);
				await fetchRunbooks(statusFilter);
			} catch (error) {
				toast.error('Failed to save runbook');
				console.error(error);
			} finally {
				setSaving(false);
			}
		},
		[editing, sopId, version, statusFilter, fetchRunbooks]
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
			label: `Active (${stats.active})`,
		},
		{
			key: 'approved',
			label: `Approved (${stats.approved})`,
		},
		{
			key: 'draft',
			label: `Drafts (${stats.draft})`,
		},
		{
			key: 'deprecated',
			label: `Deprecated (${stats.deprecated})`,
		},
	];

	return (
		<div className="runbooks-section">
			<div className="runbooks-section__header">
				<h2>Runbooks</h2>
				{canEdit && (
					<Dropdown
						menu={{
							items: [
								{ key: 'manual', label: 'Manual', onClick: handleNewRunbook },
								{ key: 'ai', label: 'AI draft from error', onClick: (): void => setAiDraftOpen(true) },
							],
						}}
					>
						<Button type="primary" icon={<PlusOutlined />}>
							New runbook
						</Button>
					</Dropdown>
				)}
			</div>

			<Tabs
				activeKey={statusFilter}
				items={tabItems}
				onChange={(key) => setStatusFilter(key as StatusFilter)}
			/>

			<Spin spinning={loading}>
				{!loading && runbooks.length === 0 && (
					<Empty description="No runbooks yet." />
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
				title={editing ? 'Edit runbook' : 'New runbook'}
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
		</div>
	);
}
