export interface CodeRcaConfig {
	contractVersion: string;
	orgId: string;
	enabled: boolean;
	minSeverity: 'critical' | 'high' | 'error' | 'warning' | 'info';
	cooldownWindowSecs: number;
	maxRunsPerDay: number;
	maxQueueDepth: number;
	maxConcurrentRuns: number;
	allowUnboundWithoutAnomaly: boolean;
	updatedAt: string;
}

export interface CodebaseRepo {
	contractVersion: string;
	orgId: string;
	repoId: string;
	gitUrl: string;
	defaultBranch: string;
	credential: string; // '<unchanged>' 센티널 = 저장된 자격증명 유지
	enabled: boolean;
	branchName: string;
	fetched: boolean;
	baselineCommit: string;
	lastSyncAt: string;
	lastSyncStatus: string;
}

export interface CodebaseServiceMap {
	orgId: string;
	serviceName: string;
	repoId: string;
	subpath: string;
}

export type CodeRcaRunStatus =
	| 'queued'
	| 'running'
	| 'done'
	| 'failed'
	| 'timeout'
	| 'unparseable';

export interface CodeRcaRunSummary {
	runId: string;
	orgId: string;
	service: string;
	status: CodeRcaRunStatus;
	baselineCommit: string;
	createdAt: number;
	finishedAt: number;
	attempts: number;
	resultRef: string;
}

export interface CodeRcaRunDetail extends CodeRcaRunSummary {
	rootCause: string;
	proposedFix: string;
	confidence: string;
	limitations: string;
}

export const CREDENTIAL_UNCHANGED = '<unchanged>';
