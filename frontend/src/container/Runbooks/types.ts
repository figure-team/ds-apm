export type RunbookStatus = 'draft' | 'approved' | 'deprecated';
export type RunbookErrorKind = 'auth' | 'timeout' | 'other';

export interface Runbook {
	id: string;
	title: string;
	description: string;
	executableScript: string;
	status: RunbookStatus;
	confidence: number;
	aiDraftedBy: string;
	sourceErrorExamples: string[];
	createdAt: string;
	updatedAt: string;
	updatedBy: string;
}

export interface DraftRunbookPayload {
	sopId: string;
	version: string;
	errorExamples: string[];
}

export interface DraftRunbookResult {
	ok: boolean;
	data?: Runbook;
	error?: string;
	errorKind?: RunbookErrorKind;
}
