import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

import type { Runbook } from 'container/Runbooks/types';

type ApiResponse<T> = {
	data: T;
	status: string;
};

const createRunbook = (
	sopId: string,
	version: string,
	runbook: Partial<Runbook>,
): Promise<ApiResponse<Runbook>> => {
	return GeneratedAPIInstance<ApiResponse<Runbook>>({
		url: `/api/v2/ds/sop/documents/${encodeURIComponent(sopId)}/versions/${encodeURIComponent(version)}/runbooks`,
		method: 'POST',
		data: runbook,
	});
};

export default createRunbook;
