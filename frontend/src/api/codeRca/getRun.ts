import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodeRcaRunDetail } from './types';

const getRun = (runId: string): Promise<AxiosResponse<CodeRcaRunDetail>> =>
	ApiV2Instance.get<CodeRcaRunDetail>(
		`/ds/coderca/runs/${encodeURIComponent(runId)}`,
	);

export default getRun;
