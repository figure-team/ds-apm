import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
	DownloadOutlined,
	PlusOutlined,
	UploadOutlined,
} from '@ant-design/icons';
import { Alert, Button, Input, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
	listSopDocuments,
	previewSopDocumentBinding,
	type SopApprovalStatus,
	type SopBindingPreviewResult,
	type SopDocumentSummary,
} from 'api/v2/rules/sopDocuments';
import RunbooksSection from 'container/Runbooks/RunbooksSection';

import './SOPDocuments.styles.scss';
import {
	downloadSopExcelTemplate,
	type ParseSopExcelResult,
} from './parseSopExcel';
import SopBulkUploadModal from './SopBulkUploadModal';
import SopBulkPreviewDrawer from './SopBulkPreviewDrawer';
import SopDocumentFormDrawer, {
	type SopDocumentEditTarget,
} from './SopDocumentFormDrawer';

function getErrorMessage(error: unknown): string {
	if (typeof error === 'object' && error !== null && 'response' in error) {
		const response = (
			error as {
				response?: { data?: { error?: string; message?: string } | string };
			}
		).response;

		if (typeof response?.data === 'string') {
			return response.data;
		}
		return response?.data?.error || response?.data?.message || 'Request failed.';
	}

	return error instanceof Error ? error.message : 'Request failed.';
}

function SOPDocuments(): JSX.Element {
	const { t } = useTranslation(['sop_documents']);

	const [documents, setDocuments] = useState<SopDocumentSummary[]>([]);
	const [bindingSopId, setBindingSopId] = useState('');
	const [bindingProjectId, setBindingProjectId] = useState('customer-a');
	const [bindingEnvironment, setBindingEnvironment] = useState('prod');
	const [bindingPreview, setBindingPreview] =
		useState<SopBindingPreviewResult>();
	const [isLoading, setIsLoading] = useState(false);
	const [isPreviewing, setIsPreviewing] = useState(false);
	const [message, setMessage] = useState('');
	const [error, setError] = useState('');
	const [uploadModalOpen, setUploadModalOpen] = useState(false);
	const [previewDrawerOpen, setPreviewDrawerOpen] = useState(false);
	const [parseResult, setParseResult] = useState<ParseSopExcelResult | null>(
		null,
	);
	const [formDrawerOpen, setFormDrawerOpen] = useState(false);
	const [formDrawerMode, setFormDrawerMode] = useState<'create' | 'edit'>(
		'create',
	);
	const [editTarget, setEditTarget] = useState<SopDocumentEditTarget>();

	const loadDocuments = useCallback(async (): Promise<void> => {
		setIsLoading(true);
		setError('');
		try {
			const response = await listSopDocuments();
			setDocuments(response.data.documents);
		} catch (requestError) {
			setError(getErrorMessage(requestError));
		} finally {
			setIsLoading(false);
		}
	}, []);

	useEffect(() => {
		void loadDocuments();
	}, [loadDocuments]);

	const openCreateDrawer = useCallback((): void => {
		setMessage('');
		setError('');
		setFormDrawerMode('create');
		setEditTarget(undefined);
		setFormDrawerOpen(true);
	}, []);

	const openEditDrawer = useCallback((record: SopDocumentSummary): void => {
		setMessage('');
		setError('');
		setFormDrawerMode('edit');
		setEditTarget({ sopId: record.sopId, version: record.version });
		setFormDrawerOpen(true);
	}, []);

	const handleFormSaved = useCallback(
		(savedMessage: string): void => {
			setMessage(savedMessage);
			void loadDocuments();
		},
		[loadDocuments],
	);

	const handleParsed = useCallback((result: ParseSopExcelResult): void => {
		setParseResult(result);
		setUploadModalOpen(false);
		setPreviewDrawerOpen(true);
	}, []);

	const handleRegistered = useCallback((): void => {
		void loadDocuments();
	}, [loadDocuments]);

	const handlePreviewBinding = useCallback(async (): Promise<void> => {
		setIsPreviewing(true);
		setBindingPreview(undefined);
		setMessage('');
		setError('');
		try {
			const response = await previewSopDocumentBinding({
				labels: {
					environment: bindingEnvironment.trim(),
					project_id: bindingProjectId.trim(),
					sop_id: bindingSopId.trim(),
				},
			});
			setBindingPreview(response.data);
		} catch (requestError) {
			setError(getErrorMessage(requestError));
		} finally {
			setIsPreviewing(false);
		}
	}, [bindingEnvironment, bindingProjectId, bindingSopId]);

	const columns = useMemo<ColumnsType<SopDocumentSummary>>(
		() => [
			{
				title: t('col_sop_id'),
				dataIndex: 'sopId',
				key: 'sopId',
			},
			{
				title: t('col_title'),
				dataIndex: 'title',
				key: 'title',
			},
			{
				title: t('col_version'),
				dataIndex: 'version',
				key: 'version',
				width: 160,
			},
			{
				title: t('col_owner'),
				dataIndex: 'ownerTeam',
				key: 'ownerTeam',
				width: 160,
			},
			{
				title: t('col_status'),
				dataIndex: 'approvalStatus',
				key: 'approvalStatus',
				width: 140,
				render: (status: SopApprovalStatus): JSX.Element => (
					<Tag color={status === 'approved' ? 'green' : 'default'}>{status}</Tag>
				),
			},
			{
				title: t('col_tenant'),
				dataIndex: 'tenantScope',
				key: 'tenantScope',
				render: (tenantScope: SopDocumentSummary['tenantScope']): JSX.Element => {
					const projectIds = tenantScope?.projectIds ?? [];
					const environments = tenantScope?.environments ?? [];
					return (
						<span>{`${projectIds.join(',')} / ${environments.join(',')}`}</span>
					);
				},
			},
			{
				title: t('col_actions'),
				key: 'actions',
				width: 96,
				render: (_, record: SopDocumentSummary): JSX.Element => (
					<Button
						data-testid="edit-sop-document"
						onClick={(): void => openEditDrawer(record)}
						size="small"
						type="link"
					>
						{t('btn_edit')}
					</Button>
				),
			},
		],
		[t, openEditDrawer],
	);

	return (
		<div className="sop-documents-page settings-shell settings-shell--narrow">
			<header className="sop-documents-page__header">
				<div className="sop-documents-page__header-row">
					<div>
						<h1>{t('page_title')}</h1>
						<p>{t('page_description')}</p>
					</div>
					<div className="sop-documents-page__header-actions">
						<Button icon={<DownloadOutlined />} onClick={downloadSopExcelTemplate}>
							{t('btn_template_download')}
						</Button>
						<Button
							icon={<UploadOutlined />}
							onClick={(): void => setUploadModalOpen(true)}
						>
							{t('btn_file_upload')}
						</Button>
						<Button
							data-testid="open-register-drawer"
							icon={<PlusOutlined />}
							onClick={openCreateDrawer}
							type="primary"
						>
							{t('btn_add_document')}
						</Button>
					</div>
				</div>
			</header>

			{message && <Alert message={message} showIcon type="success" />}
			{error && <Alert message={error} showIcon type="error" />}

			<section className="sop-documents-page__section">
				<div className="sop-documents-page__section-header">
					<h2>{t('documents_section_title')}</h2>
					<p>{t('documents_section_description')}</p>
				</div>
				<Table
					columns={columns}
					dataSource={documents}
					expandable={{
						expandedRowRender: (record: SopDocumentSummary): JSX.Element => (
							<RunbooksSection sopId={record.sopId} version={record.version} />
						),
						rowExpandable: (record: SopDocumentSummary): boolean =>
							Boolean(record.sopId && record.version),
					}}
					loading={isLoading}
					pagination={false}
					rowKey={(document): string => `${document.sopId}:${document.version}`}
					size="small"
				/>
			</section>

			<section className="sop-documents-page__section">
				<div className="sop-documents-page__section-header">
					<h2>{t('binding_section_title')}</h2>
					<p>{t('binding_section_description')}</p>
				</div>
				<div className="sop-documents-page__binding">
					<Input
						data-testid="binding-sop-id"
						onChange={(event): void => setBindingSopId(event.target.value)}
						placeholder="SOP-PAY-001"
						value={bindingSopId}
					/>
					<Input
						data-testid="binding-project-id"
						onChange={(event): void => setBindingProjectId(event.target.value)}
						placeholder="customer-a"
						value={bindingProjectId}
					/>
					<Input
						data-testid="binding-environment"
						onChange={(event): void => setBindingEnvironment(event.target.value)}
						placeholder="prod"
						value={bindingEnvironment}
					/>
					<Button
						data-testid="preview-sop-binding"
						disabled={
							!bindingSopId.trim() ||
							!bindingProjectId.trim() ||
							!bindingEnvironment.trim()
						}
						loading={isPreviewing}
						onClick={handlePreviewBinding}
					>
						{t('btn_preview_binding')}
					</Button>
				</div>
				{bindingPreview && (
					<div className="sop-documents-page__binding-result">
						<Tag color={bindingPreview.status === 'bound' ? 'green' : 'orange'}>
							{bindingPreview.status}
						</Tag>
						<span>{bindingPreview.resolution}</span>
						<span>{bindingPreview.sopId || bindingSopId}</span>
						{bindingPreview.title && <span>{bindingPreview.title}</span>}
						{bindingPreview.version && <span>{bindingPreview.version}</span>}
						{bindingPreview.warnings?.map((warning) => (
							<span className="sop-documents-page__warning" key={warning}>
								{warning}
							</span>
						))}
					</div>
				)}
			</section>

			<SopDocumentFormDrawer
				editTarget={editTarget}
				mode={formDrawerMode}
				onClose={(): void => setFormDrawerOpen(false)}
				onSaved={handleFormSaved}
				open={formDrawerOpen}
			/>
			<SopBulkUploadModal
				onClose={(): void => setUploadModalOpen(false)}
				onParsed={handleParsed}
				open={uploadModalOpen}
			/>
			<SopBulkPreviewDrawer
				onClose={(): void => setPreviewDrawerOpen(false)}
				onRegistered={handleRegistered}
				open={previewDrawerOpen}
				parseResult={parseResult}
			/>
		</div>
	);
}

export default SOPDocuments;
