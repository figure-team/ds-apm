import { ApiV2Instance } from 'api';

import { ApiEnvelope } from './types';

interface ExportRunResult {
	path: string;
}

const exportRun = async (runId: string): Promise<{ data: ExportRunResult }> => {
	const res = await ApiV2Instance.post<ApiEnvelope<ExportRunResult>>(
		`/ds/coderca/runs/${encodeURIComponent(runId)}/export`,
	);
	return { data: res.data.data };
};

export default exportRun;
