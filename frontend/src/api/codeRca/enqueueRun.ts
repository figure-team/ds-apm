import { ApiV2Instance } from 'api';

import { ApiEnvelope } from './types';

export interface EnqueueRunResult {
	admitted: boolean;
	runId: string;
	reason: string;
}

const enqueueRun = async (service: string): Promise<EnqueueRunResult> => {
	const res = await ApiV2Instance.post<ApiEnvelope<EnqueueRunResult>>(
		'/ds/coderca/runs',
		{ service },
	);
	return res.data.data;
};

export default enqueueRun;
