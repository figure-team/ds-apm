import { ApiV2Instance } from 'api';

import { ApiEnvelope, IncidentReportTemplate } from './types';

const getTemplate = async (): Promise<IncidentReportTemplate> => {
	const res = await ApiV2Instance.get<ApiEnvelope<IncidentReportTemplate>>(
		'/ds/incident/report/template',
	);
	return res.data.data;
};

export default getTemplate;
