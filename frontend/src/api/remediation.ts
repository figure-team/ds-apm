import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

export interface RemediationExecution {
	id: string;
	status: string;
	scriptSnapshot: string;
	sopId: string;
	runbookId: string;
	incidentId?: string;
	proposedAt?: string;
	approvedAt?: string;
	executedAt?: string;
	terminalAt?: string;
	approvedBy?: string;
	expiresAt?: string;
	exitCode?: number;
	outputSnippet?: string;
	verifyResult?: string;
}

type ApiResponse<T> = {
	data: T;
	status: string;
};

export const getRemediation = (id: string): Promise<RemediationExecution> =>
	GeneratedAPIInstance<ApiResponse<RemediationExecution>>({
		url: `/api/v2/ds/remediation/${encodeURIComponent(id)}`,
		method: 'GET',
	}).then((r) => r.data);

export const approveRemediation = (id: string): Promise<RemediationExecution> =>
	GeneratedAPIInstance<ApiResponse<RemediationExecution>>({
		url: `/api/v2/ds/remediation/${encodeURIComponent(id)}/approve`,
		method: 'POST',
	}).then((r) => r.data);

export const rejectRemediation = (id: string): Promise<RemediationExecution> =>
	GeneratedAPIInstance<ApiResponse<RemediationExecution>>({
		url: `/api/v2/ds/remediation/${encodeURIComponent(id)}/reject`,
		method: 'POST',
	}).then((r) => r.data);

export interface ListRemediationsParams {
	status?: string;
	sopId?: string;
	limit?: number;
}

export const listRemediations = (
	params: ListRemediationsParams = {},
): Promise<RemediationExecution[]> => {
	const search = new URLSearchParams({ scope: 'org' });
	if (params.status) search.set('status', params.status);
	if (params.sopId) search.set('sopId', params.sopId);
	if (params.limit) search.set('limit', String(params.limit));
	return GeneratedAPIInstance<ApiResponse<{ remediations: RemediationExecution[] }>>({
		url: `/api/v2/ds/remediation?${search.toString()}`,
		method: 'GET',
	}).then((r) => r.data.remediations);
};

// RemediationConfig is the org-wide auto-remediation master switch + timing
// knobs. Admin-only on both read and write (the backend routes enforce
// AdminAccess). executionEnabled is the toggle surfaced on the SOP page.
export interface RemediationConfig {
	executionEnabled: boolean;
	proposalTtlSeconds: number;
	execTimeoutSeconds: number;
	verifyWindowSeconds: number;
	maxConcurrent: number;
}

export const getRemediationConfig = (): Promise<RemediationConfig> =>
	GeneratedAPIInstance<ApiResponse<RemediationConfig>>({
		url: `/api/v2/ds/remediation/config`,
		method: 'GET',
	}).then((r) => r.data);

export const updateRemediationConfig = (
	config: RemediationConfig,
): Promise<RemediationConfig> =>
	GeneratedAPIInstance<ApiResponse<RemediationConfig>>({
		url: `/api/v2/ds/remediation/config`,
		method: 'PUT',
		data: config,
	}).then((r) => r.data);
