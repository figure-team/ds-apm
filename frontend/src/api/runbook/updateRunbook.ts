import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

import type { Runbook } from 'container/Runbooks/types';

type ApiResponse<T> = {
	data: T;
	status: string;
};

const updateRunbook = (
	sopId: string,
	version: string,
	runbook: Runbook,
): Promise<ApiResponse<Runbook>> => {
	return GeneratedAPIInstance<ApiResponse<Runbook>>({
		url: `/api/v2/ds/sop/documents/${encodeURIComponent(sopId)}/versions/${encodeURIComponent(version)}/runbooks/${encodeURIComponent(runbook.id)}`,
		method: 'PUT',
		data: runbook,
	});
};

export default updateRunbook;
