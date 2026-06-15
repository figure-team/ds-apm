import { ApiV2Instance } from 'api';

import {
	ApiEnvelope,
	GenerateReportRequest,
	GenerateReportResult,
} from './types';

const generateReport = async (
	body: GenerateReportRequest,
): Promise<GenerateReportResult> => {
	const res = await ApiV2Instance.post<ApiEnvelope<GenerateReportResult>>(
		'/ds/incident/report',
		body,
	);
	return res.data.data;
};

export default generateReport;
