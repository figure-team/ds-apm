import { ApiV2Instance } from 'api';

import { ApiEnvelope, CodeRcaRunSummary } from './types';

export interface ListRunsParams {
	status?: string;
	service?: string;
	limit?: number;
	offset?: number;
}

const listRuns = async (
	params: ListRunsParams,
): Promise<{ data: CodeRcaRunSummary[] }> => {
	const res = await ApiV2Instance.get<ApiEnvelope<CodeRcaRunSummary[]>>(
		'/ds/coderca/runs',
		{ params },
	);
	return { data: res.data.data };
};

export default listRuns;
