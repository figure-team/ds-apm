export interface ApiEnvelope<T> {
	status: string;
	data: T;
}

export interface IncidentReportTemplate {
	template: string;
	isDefault: boolean;
}

export interface GenerateReportRequest {
	incidentId: string;
	alertFingerprint?: string;
	service?: string;
	severity?: string;
}

export interface GenerateReportResult {
	markdown: string;
	// The structured report is returned too; kept loose here since the UI
	// renders the markdown.
	report: Record<string, unknown>;
}
