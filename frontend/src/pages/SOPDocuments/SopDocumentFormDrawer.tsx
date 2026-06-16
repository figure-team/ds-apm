import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Drawer, Input, Select, Spin } from 'antd';
import SHA256 from 'crypto-js/sha256';
import {
	createSopDocument,
	getSopDocument,
	SOP_DOCUMENT_CONTRACT_VERSION,
	type SopApprovalStatus,
	type SopDocument,
} from 'api/v2/rules/sopDocuments';

export type SopDocumentFormState = {
	sopId: string;
	title: string;
	version: string;
	sourceId: string;
	bodyMarkdown: string;
	customerUpdateTemplate: string;
	vendorRequestTemplate: string;
	displayUrl: string;
	ownerTeam: string;
	approvalStatus: SopApprovalStatus;
	projectIds: string;
	environments: string;
	tags: string;
	serviceAccountProfile: string;
};

const DEFAULT_FORM_STATE: SopDocumentFormState = {
	sopId: '',
	title: '',
	version: '',
	sourceId: 'src-managed-markdown-default',
	bodyMarkdown: '',
	customerUpdateTemplate: '',
	vendorRequestTemplate: '',
	displayUrl: '',
	ownerTeam: '',
	approvalStatus: 'approved',
	projectIds: 'customer-a',
	environments: 'prod',
	tags: '',
	serviceAccountProfile: 'managed-markdown-local',
};

function parseTags(value: string): string[] {
	return value
		.split(',')
		.map((tag) => tag.trim())
		.filter(Boolean);
}

function checksumForMarkdown(bodyMarkdown: string): string {
	return `sha256:${SHA256(bodyMarkdown).toString()}`;
}

function buildSopDocument(form: SopDocumentFormState): SopDocument {
	return {
		contractVersion: SOP_DOCUMENT_CONTRACT_VERSION,
		sopId: form.sopId.trim(),
		title: form.title.trim(),
		version: form.version.trim(),
		checksum: checksumForMarkdown(form.bodyMarkdown),
		source: {
			type: 'managed_markdown',
			sourceId: form.sourceId.trim(),
		},
		bodyMarkdown: form.bodyMarkdown,
		customerUpdateTemplate: form.customerUpdateTemplate.trim() || undefined,
		vendorRequestTemplate: form.vendorRequestTemplate.trim() || undefined,
		displayUrl: form.displayUrl.trim() || undefined,
		ownerTeam: form.ownerTeam.trim(),
		approvalStatus: form.approvalStatus,
		tenantScope: {
			projectIds: parseTags(form.projectIds),
			environments: parseTags(form.environments),
		},
		tags: parseTags(form.tags),
		updatedAt: new Date().toISOString(),
		securityContext: {
			serviceAccountProfile: form.serviceAccountProfile.trim(),
			secretRefVisible: false,
			browserCredentialsUsed: false,
			redactionApplied: true,
		},
	};
}

function toFormState(doc: SopDocument): SopDocumentFormState {
	return {
		sopId: doc.sopId,
		title: doc.title,
		version: doc.version,
		sourceId: doc.source.sourceId,
		bodyMarkdown: doc.bodyMarkdown,
		customerUpdateTemplate: doc.customerUpdateTemplate ?? '',
		vendorRequestTemplate: doc.vendorRequestTemplate ?? '',
		displayUrl: doc.displayUrl ?? '',
		ownerTeam: doc.ownerTeam,
		approvalStatus: doc.approvalStatus,
		projectIds: (doc.tenantScope?.projectIds ?? []).join(', '),
		environments: (doc.tenantScope?.environments ?? []).join(', '),
		tags: (doc.tags ?? []).join(', '),
		serviceAccountProfile: doc.securityContext?.serviceAccountProfile ?? '',
	};
}

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

function isFormIncomplete(form: SopDocumentFormState): boolean {
	return !(
		form.sopId.trim() &&
		form.title.trim() &&
		form.version.trim() &&
		form.sourceId.trim() &&
		form.bodyMarkdown.trim() &&
		form.ownerTeam.trim() &&
		form.projectIds.trim() &&
		form.environments.trim() &&
		form.serviceAccountProfile.trim()
	);
}

export type SopDocumentEditTarget = {
	sopId: string;
	version: string;
};

type Props = {
	open: boolean;
	mode: 'create' | 'edit';
	editTarget?: SopDocumentEditTarget;
	onClose: () => void;
	onSaved: (message: string) => void;
};

function SopDocumentFormDrawer({
	open,
	mode,
	editTarget,
	onClose,
	onSaved,
}: Props): JSX.Element {
	const { t } = useTranslation(['sop_documents']);

	const [form, setForm] = useState<SopDocumentFormState>(DEFAULT_FORM_STATE);
	const [originalDoc, setOriginalDoc] = useState<SopDocument>();
	const [isLoading, setIsLoading] = useState(false);
	const [isSaving, setIsSaving] = useState(false);
	const [error, setError] = useState('');

	const approvalStatusOptions = useMemo<
		{ label: string; value: SopApprovalStatus }[]
	>(
		() => [
			{ label: t('status_approved'), value: 'approved' },
			{ label: t('status_draft'), value: 'draft' },
			{ label: t('status_deprecated'), value: 'deprecated' },
			{ label: t('status_disabled'), value: 'disabled' },
		],
		[t],
	);

	const editSopId = editTarget?.sopId;
	const editVersion = editTarget?.version;

	// Populate the form whenever the drawer opens. Edit mode fetches the full
	// document (the table only holds a summary without bodyMarkdown).
	useEffect(() => {
		if (!open) {
			return undefined;
		}
		setError('');
		if (mode === 'create' || !editSopId || !editVersion) {
			setForm(DEFAULT_FORM_STATE);
			setOriginalDoc(undefined);
			return undefined;
		}

		let cancelled = false;
		setIsLoading(true);
		getSopDocument(editSopId, editVersion)
			.then((response) => {
				if (cancelled) {
					return;
				}
				setOriginalDoc(response.data);
				setForm(toFormState(response.data));
			})
			.catch((requestError) => {
				if (!cancelled) {
					setError(getErrorMessage(requestError));
				}
			})
			.finally(() => {
				if (!cancelled) {
					setIsLoading(false);
				}
			});

		return (): void => {
			cancelled = true;
		};
	}, [open, mode, editSopId, editVersion]);

	const handleFieldChange = useCallback(
		<Key extends keyof SopDocumentFormState>(
			key: Key,
			value: SopDocumentFormState[Key],
		): void => {
			setForm((prev) => ({ ...prev, [key]: value }));
			setError('');
		},
		[],
	);

	// A안: editing publishes a new version. The new version must differ from the
	// one being edited so the previous version is preserved (then deprecated).
	const versionUnchanged =
		mode === 'edit' &&
		Boolean(originalDoc) &&
		form.version.trim() === originalDoc?.version;

	const submitDisabled =
		isLoading || isSaving || isFormIncomplete(form) || versionUnchanged;

	const handleSave = useCallback(async (): Promise<void> => {
		setIsSaving(true);
		setError('');
		try {
			const document = buildSopDocument(form);
			await createSopDocument(document);

			// A안: publishing a new version deprecates the previous one.
			if (
				mode === 'edit' &&
				originalDoc &&
				originalDoc.version !== document.version
			) {
				await createSopDocument({
					...originalDoc,
					approvalStatus: 'deprecated',
					updatedAt: new Date().toISOString(),
				});
			}

			onSaved(
				`${t('msg_save_success_prefix')} ${document.sopId} ${document.version}`,
			);
			onClose();
		} catch (requestError) {
			setError(getErrorMessage(requestError));
		} finally {
			setIsSaving(false);
		}
	}, [form, mode, originalDoc, onSaved, onClose, t]);

	const isEdit = mode === 'edit';

	return (
		<Drawer
			data-testid="sop-document-form-drawer"
			destroyOnClose
			onClose={onClose}
			open={open}
			title={isEdit ? t('drawer_edit_title') : t('drawer_create_title')}
			width={820}
			footer={
				<div className="sop-documents-page__drawer-footer">
					<Button onClick={onClose}>{t('btn_cancel')}</Button>
					<Button
						data-testid="register-sop-document"
						disabled={submitDisabled}
						loading={isSaving}
						onClick={handleSave}
						type="primary"
					>
						{isEdit ? t('btn_save_new_version') : t('btn_register')}
					</Button>
				</div>
			}
		>
			{error && <Alert message={error} showIcon type="error" />}
			{isEdit && (
				<Alert
					message={t('edit_version_hint')}
					showIcon
					type="info"
					style={{ marginBottom: 'var(--spacing-8)' }}
				/>
			)}
			{isLoading ? (
				<Spin />
			) : (
				<div className="sop-documents-page__form-grid">
					<label htmlFor="sop-document-sop-id-input">
						<span>{t('field_sop_id')}</span>
						<Input
							data-testid="sop-document-sop-id"
							disabled={isEdit}
							id="sop-document-sop-id-input"
							onChange={(event): void =>
								handleFieldChange('sopId', event.target.value)
							}
							placeholder="SOP-PAY-001"
							value={form.sopId}
						/>
					</label>
					<label htmlFor="sop-document-title-input">
						<span>{t('field_title')}</span>
						<Input
							data-testid="sop-document-title"
							id="sop-document-title-input"
							onChange={(event): void =>
								handleFieldChange('title', event.target.value)
							}
							placeholder="Payment API 5xx response"
							value={form.title}
						/>
					</label>
					<label htmlFor="sop-document-version-input">
						<span>{t('field_version')}</span>
						<Input
							data-testid="sop-document-version"
							id="sop-document-version-input"
							onChange={(event): void =>
								handleFieldChange('version', event.target.value)
							}
							placeholder="2026-05-12.1"
							status={versionUnchanged ? 'error' : undefined}
							value={form.version}
						/>
						{versionUnchanged && (
							<span className="sop-documents-page__field-error">
								{t('edit_version_same_error')}
							</span>
						)}
					</label>
					<label htmlFor="sop-document-owner-team-input">
						<span>{t('field_owner_team')}</span>
						<Input
							data-testid="sop-document-owner-team"
							id="sop-document-owner-team-input"
							onChange={(event): void =>
								handleFieldChange('ownerTeam', event.target.value)
							}
							placeholder="payments"
							value={form.ownerTeam}
						/>
					</label>
					<label htmlFor="sop-document-approval-status-input">
						<span>{t('field_approval_status')}</span>
						<Select
							id="sop-document-approval-status-input"
							options={approvalStatusOptions}
							onChange={(value): void =>
								handleFieldChange('approvalStatus', value)
							}
							value={form.approvalStatus}
						/>
					</label>
					<label htmlFor="sop-document-source-id-input">
						<span>{t('field_source_id')}</span>
						<Input
							id="sop-document-source-id-input"
							onChange={(event): void =>
								handleFieldChange('sourceId', event.target.value)
							}
							value={form.sourceId}
						/>
					</label>
					<label htmlFor="sop-document-project-ids-input">
						<span>{t('field_project_ids')}</span>
						<Input
							data-testid="sop-document-project-ids"
							id="sop-document-project-ids-input"
							onChange={(event): void =>
								handleFieldChange('projectIds', event.target.value)
							}
							placeholder="customer-a"
							value={form.projectIds}
						/>
					</label>
					<label htmlFor="sop-document-environments-input">
						<span>{t('field_environments')}</span>
						<Input
							data-testid="sop-document-environments"
							id="sop-document-environments-input"
							onChange={(event): void =>
								handleFieldChange('environments', event.target.value)
							}
							placeholder="prod"
							value={form.environments}
						/>
					</label>
					<label htmlFor="sop-document-display-url-input">
						<span>{t('field_display_url')}</span>
						<Input
							id="sop-document-display-url-input"
							onChange={(event): void =>
								handleFieldChange('displayUrl', event.target.value)
							}
							placeholder="https://kb.example/sop/SOP-PAY-001"
							value={form.displayUrl}
						/>
					</label>
					<label htmlFor="sop-document-tags-input">
						<span>{t('field_tags')}</span>
						<Input
							id="sop-document-tags-input"
							onChange={(event): void =>
								handleFieldChange('tags', event.target.value)
							}
							placeholder="payment-api, critical"
							value={form.tags}
						/>
					</label>
					<label htmlFor="sop-document-service-account-profile-input">
						<span>{t('field_service_account_profile')}</span>
						<Input
							id="sop-document-service-account-profile-input"
							onChange={(event): void =>
								handleFieldChange('serviceAccountProfile', event.target.value)
							}
							value={form.serviceAccountProfile}
						/>
					</label>
					<label
						className="sop-documents-page__markdown"
						htmlFor="sop-document-body-markdown-input"
					>
						<span>{t('field_body_markdown')}</span>
						<Input.TextArea
							data-testid="sop-document-body-markdown"
							id="sop-document-body-markdown-input"
							onChange={(event): void =>
								handleFieldChange('bodyMarkdown', event.target.value)
							}
							placeholder={
								'# Payment API 5xx response\n\n1. Check payment success dashboard\n2. Inspect PG timeout logs'
							}
							rows={7}
							value={form.bodyMarkdown}
						/>
					</label>
					<label
						className="sop-documents-page__markdown"
						htmlFor="sop-document-customer-update-template-input"
					>
						<span>{t('field_customer_update_template')}</span>
						<Input.TextArea
							data-testid="sop-document-customer-update-template"
							id="sop-document-customer-update-template-input"
							onChange={(event): void =>
								handleFieldChange('customerUpdateTemplate', event.target.value)
							}
							placeholder={
								'[○○ 서비스 이용 안내]\n\n■ 발생 현황: {현재 상황}\n■ 영향 범위: {영향 범위}\n■ 조치 사항: {조치}\n■ 향후 안내: {다음 안내}\n■ 문의처: 고객센터 1588-0000'
							}
							rows={6}
							value={form.customerUpdateTemplate}
						/>
					</label>
					<label
						className="sop-documents-page__markdown"
						htmlFor="sop-document-vendor-request-template-input"
					>
						<span>{t('field_vendor_request_template')}</span>
						<Input.TextArea
							data-testid="sop-document-vendor-request-template"
							id="sop-document-vendor-request-template-input"
							onChange={(event): void =>
								handleFieldChange('vendorRequestTemplate', event.target.value)
							}
							placeholder={
								'안녕하세요. {서비스}에서 {증상}이 확인되었습니다. {확인 요청 항목} 확인 부탁드립니다.'
							}
							rows={4}
							value={form.vendorRequestTemplate}
						/>
					</label>
				</div>
			)}
		</Drawer>
	);
}

export default SopDocumentFormDrawer;
