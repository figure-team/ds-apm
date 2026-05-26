import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

import type { Runbook, DraftRunbookPayload, DraftRunbookResult } from 'container/Runbooks/types';

type ApiResponse<T> = {
	data: T;
	status: string;
};

type DraftRunbookResponse = Runbook | DraftRunbookResult;

const draftRunbook = (
	payload: DraftRunbookPayload,
): Promise<ApiResponse<DraftRunbookResponse>> => {
	return GeneratedAPIInstance<ApiResponse<DraftRunbookResponse>>({
		url: '/api/v2/ds/runbooks/draft',
		method: 'POST',
		data: payload,
	});
};

export default draftRunbook;
