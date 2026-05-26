import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

import type { Runbook } from 'container/Runbooks/types';

export interface ListRunbooksResponse {
	runbooks: Runbook[];
}

type ApiResponse<T> = {
	data: T;
	status: string;
};

const listRunbooks = (
	sopId: string,
	version: string,
	statusFilter?: string,
): Promise<ApiResponse<ListRunbooksResponse>> => {
	const query = statusFilter
		? `?status=${encodeURIComponent(statusFilter)}`
		: '';
	return GeneratedAPIInstance<ApiResponse<ListRunbooksResponse>>({
		url: `/api/v2/ds/sop/documents/${encodeURIComponent(sopId)}/versions/${encodeURIComponent(version)}/runbooks${query}`,
		method: 'GET',
	});
};

export default listRunbooks;
