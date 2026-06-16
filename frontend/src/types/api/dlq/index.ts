export type DLQStatus = 'pending' | 'replayed' | 'replay_failed';

export interface DLQEntry {
	event_id: string;
	channel: string;
	failed_at: string; // ISO 8601
	reason: string;
	status: DLQStatus;
	payload: string; // base64-encoded JSON
}

export interface ReplayResult {
	replayed: number;
	skipped: number;
	failed: number;
}

export interface GetDLQEntriesParams {
	channel?: string;
	status?: DLQStatus | '';
}

export interface ReplayDLQEntriesPayload {
	event_ids: string[];
}
