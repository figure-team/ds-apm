import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

import type { Runbook } from 'container/Runbooks/types';

type ApiResponse<T> = {
	data: T;
	status: string;
};

const getRunbook = (
	sopId: string,
	version: string,
	runbookId: string,
): Promise<ApiResponse<Runbook>> => {
	return GeneratedAPIInstance<ApiResponse<Runbook>>({
		url: `/api/v2/ds/sop/documents/${encodeURIComponent(sopId)}/versions/${encodeURIComponent(version)}/runbooks/${encodeURIComponent(runbookId)}`,
		method: 'GET',
	});
};

export default getRunbook;
