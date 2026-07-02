import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

// RemediationTargetWire is the read shape of /api/v2/ds/remediation/targets.
// The sealed credential is never present; hasCredential drives the edit form's
// "keep existing key" default (design §3.5).
export interface RemediationTargetWire {
	id: string;
	orgId: string;
	name: string;
	host: string;
	port: number;
	user: string;
	credentialKind: string;
	hostKeyFingerprint: string;
	serviceSelectors: string[];
	hasCredential: boolean;
	createdAt: string;
	updatedAt: string;
}

// Exactly one of privateKeyPEM / sealedPrivateKey must be set (design §3.2).
export interface RemediationTargetCredential {
	kind: 'private_key';
	privateKeyPEM?: string;
	sealedPrivateKey?: string;
}

export interface RemediationTargetUpsert {
	name: string;
	host: string;
	port: number;
	user: string;
	serviceSelectors: string[];
	hostKeyFingerprint: string;
	// Omit on edit to keep the stored key (design §3.2).
	credential?: RemediationTargetCredential;
}

export interface RemediationTargetListResponse {
	targets: RemediationTargetWire[];
	encryptionReady: boolean;
}

export interface KeygenResponse {
	publicKeyOpenSSH: string;
	sealedPrivateKey: string;
}

export interface FingerprintResponse {
	fingerprint: string;
	keyType: string;
}

// Either targetId (stored credential) or draft connection params (design §3.1).
export interface ConnectionTestRequest {
	targetId?: string;
	host?: string;
	port?: number;
	user?: string;
	hostKeyFingerprint?: string;
	credential?: RemediationTargetCredential;
}

export interface ConnectionTestResult {
	ok: boolean;
	exitCode?: number;
	output?: string;
	error?: string;
}

type ApiResponse<T> = {
	data: T;
	status: string;
};

const BASE = '/api/v2/ds/remediation/targets';

export const listRemediationTargets = (): Promise<RemediationTargetListResponse> =>
	GeneratedAPIInstance<ApiResponse<RemediationTargetListResponse>>({
		url: BASE,
		method: 'GET',
	}).then((r) => r.data);

export const createRemediationTarget = (
	body: RemediationTargetUpsert,
): Promise<RemediationTargetWire> =>
	GeneratedAPIInstance<ApiResponse<RemediationTargetWire>>({
		url: BASE,
		method: 'POST',
		data: body,
	}).then((r) => r.data);

export const updateRemediationTarget = (
	id: string,
	body: RemediationTargetUpsert,
): Promise<RemediationTargetWire> =>
	GeneratedAPIInstance<ApiResponse<RemediationTargetWire>>({
		url: `${BASE}/${encodeURIComponent(id)}`,
		method: 'PUT',
		data: body,
	}).then((r) => r.data);

export const deleteRemediationTarget = (id: string): Promise<void> =>
	GeneratedAPIInstance<ApiResponse<unknown>>({
		url: `${BASE}/${encodeURIComponent(id)}`,
		method: 'DELETE',
	}).then(() => undefined);

export const generateRemediationKeyPair = (): Promise<KeygenResponse> =>
	GeneratedAPIInstance<ApiResponse<KeygenResponse>>({
		url: `${BASE}/keygen`,
		method: 'POST',
	}).then((r) => r.data);

export const fetchHostKeyFingerprint = (
	host: string,
	port: number,
): Promise<FingerprintResponse> =>
	GeneratedAPIInstance<ApiResponse<FingerprintResponse>>({
		url: `${BASE}/fingerprint`,
		method: 'POST',
		data: { host, port },
	}).then((r) => r.data);

export const testRemediationConnection = (
	body: ConnectionTestRequest,
): Promise<ConnectionTestResult> =>
	GeneratedAPIInstance<ApiResponse<ConnectionTestResult>>({
		url: `${BASE}/test`,
		method: 'POST',
		data: body,
	}).then((r) => r.data);
