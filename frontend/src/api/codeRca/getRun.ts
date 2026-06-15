import { ApiV2Instance } from 'api';

import { ApiEnvelope, CodeRcaRunDetail } from './types';

const getRun = async (runId: string): Promise<{ data: CodeRcaRunDetail }> => {
	const res = await ApiV2Instance.get<ApiEnvelope<CodeRcaRunDetail>>(
		`/ds/coderca/runs/${encodeURIComponent(runId)}`,
	);
	return { data: res.data.data };
};

export default getRun;
