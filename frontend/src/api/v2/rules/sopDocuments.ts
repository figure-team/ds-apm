import { GeneratedAPIInstance } from 'api/generatedAPIInstance';
import type { Labels } from 'types/api/alerts/def';

export const SOP_DOCUMENT_CONTRACT_VERSION = 'ds.sop_document.v1';
export const SOP_DOCUMENT_LIST_CONTRACT_VERSION = 'ds.sop_document_list.v1';
export const SOP_BINDING_CONTRACT_VERSION = 'ds.sop_binding.v1';

export type SopApprovalStatus =
	| 'draft'
	| 'approved'
	| 'deprecated'
	| 'disabled';

export type SopDocumentSource = {
	type: 'managed_markdown';
	sourceId: string;
};

export type SopDocumentSecurityContext = {
	serviceAccountProfile: string;
	secretRefVisible: boolean;
	browserCredentialsUsed: boolean;
	redactionApplied: boolean;
};

export type SopTenantScope = {
	projectIds: string[];
	environments: string[];
};

export type SopDocument = {
	contractVersion: typeof SOP_DOCUMENT_CONTRACT_VERSION;
	sopId: string;
	title: string;
	version: string;
	checksum: string;
	source: SopDocumentSource;
	bodyMarkdown: string;
	// Optional org-approved comms templates filled by the AI generator (CF-2).
	customerUpdateTemplate?: string;
	vendorRequestTemplate?: string;
	displayUrl?: string;
	ownerTeam: string;
	approvalStatus: SopApprovalStatus;
	tenantScope: SopTenantScope;
	tags?: string[];
	updatedAt: string;
	securityContext: SopDocumentSecurityContext;
};

export type SopDocumentSummary = Omit<
	SopDocument,
	'bodyMarkdown' | 'securityContext'
>;

export type SopDocumentListResult = {
	contractVersion: typeof SOP_DOCUMENT_LIST_CONTRACT_VERSION;
	documents: SopDocumentSummary[];
};

export type SopBindingPreviewRequest = {
	labels?: Labels;
	annotations?: Labels;
};

export type SopBindingPreviewResult = {
	contractVersion: typeof SOP_BINDING_CONTRACT_VERSION;
	status: 'bound' | 'missing' | 'disabled' | 'forbidden';
	resolution: 'explicit_label' | 'no_match';
	sopId?: string;
	version?: string;
	title?: string;
	sourceId?: string;
	warnings?: string[];
};

type ApiResponse<T> = {
	data: T;
	status: string;
};

export function createSopDocument(
	data: SopDocument,
): Promise<ApiResponse<SopDocument>> {
	return GeneratedAPIInstance<ApiResponse<SopDocument>>({
		url: '/api/v2/ds/sop/documents',
		method: 'POST',
		data,
	});
}

export function listSopDocuments(): Promise<
	ApiResponse<SopDocumentListResult>
> {
	return GeneratedAPIInstance<ApiResponse<SopDocumentListResult>>({
		url: '/api/v2/ds/sop/documents',
		method: 'GET',
	});
}

export function getSopDocument(
	sopId: string,
	version: string,
): Promise<ApiResponse<SopDocument>> {
	return GeneratedAPIInstance<ApiResponse<SopDocument>>({
		url: `/api/v2/ds/sop/documents/${encodeURIComponent(
			sopId,
		)}/versions/${encodeURIComponent(version)}`,
		method: 'GET',
	});
}

export function previewSopDocumentBinding(
	data: SopBindingPreviewRequest,
): Promise<ApiResponse<SopBindingPreviewResult>> {
	return GeneratedAPIInstance<ApiResponse<SopBindingPreviewResult>>({
		url: '/api/v2/ds/sop/bindings/preview',
		method: 'POST',
		data,
	});
}

export const SOP_BATCH_RESULT_CONTRACT_VERSION = 'ds.sop_batch_result.v1';

export type SopDocumentBatchRequest = {
	contractVersion: string;
	documents: SopDocument[];
};

export type SopDocumentBatchResult = {
	sopId: string;
	version: string;
	status: 'ok' | 'error';
	error?: string;
};

export type SopDocumentBatchResponse = {
	contractVersion: string;
	total: number;
	succeeded: number;
	failed: number;
	results: SopDocumentBatchResult[];
};

export function createSopDocumentBatch(
	data: SopDocumentBatchRequest,
): Promise<ApiResponse<SopDocumentBatchResponse>> {
	return GeneratedAPIInstance<ApiResponse<SopDocumentBatchResponse>>({
		url: '/api/v2/ds/sop/documents/batch',
		method: 'POST',
		data,
	});
}
