import { ApiV2Instance } from 'api';

import { ApiEnvelope, CodeRcaConfig } from './types';

const getConfig = async (): Promise<{ data: CodeRcaConfig }> => {
	const res = await ApiV2Instance.get<ApiEnvelope<CodeRcaConfig>>(
		'/ds/coderca/config',
	);
	return { data: res.data.data };
};

export default getConfig;
