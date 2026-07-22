export interface CodeRcaConfig {
	contractVersion: string;
	orgId: string;
	enabled: boolean;
	minSeverity: 'critical' | 'error' | 'warning' | 'info';
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
	// 운영 프로젝트 로컬 루트. 설정 시 <경로>/ds-hub/에 RCA 산출물 저장.
	artifactPath: string;
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
	failureReason: string; // why a non-done run ended; '' for done
}

export interface CodeRcaRunDetail extends CodeRcaRunSummary {
	rootCause: string;
	proposedFix: string;
	confidence: string;
	limitations: string;
}

export const CREDENTIAL_UNCHANGED = '<unchanged>';

// Backend wraps every render.Success response in { status, data }.
// Read clients unwrap one level so consumers receive the bare payload.
export interface ApiEnvelope<T> {
	status: string;
	data: T;
}
