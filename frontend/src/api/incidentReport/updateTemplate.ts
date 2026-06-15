import { ApiV2Instance } from 'api';

import { ApiEnvelope, IncidentReportTemplate } from './types';

const updateTemplate = async (
	template: string,
): Promise<IncidentReportTemplate> => {
	const res = await ApiV2Instance.put<ApiEnvelope<IncidentReportTemplate>>(
		'/ds/incident/report/template',
		{ template },
	);
	return res.data.data;
};

export default updateTemplate;
