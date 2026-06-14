import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodeRcaRunSummary } from './types';

export interface ListRunsParams {
	status?: string;
	service?: string;
	limit?: number;
	offset?: number;
}

const listRuns = (
	params: ListRunsParams,
): Promise<AxiosResponse<CodeRcaRunSummary[]>> =>
	ApiV2Instance.get<CodeRcaRunSummary[]>('/ds/coderca/runs', { params });

export default listRuns;
