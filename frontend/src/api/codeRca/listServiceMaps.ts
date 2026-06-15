import { ApiV2Instance } from 'api';

import { ApiEnvelope, CodebaseServiceMap } from './types';

const listServiceMaps = async (): Promise<{ data: CodebaseServiceMap[] }> => {
	const res = await ApiV2Instance.get<ApiEnvelope<CodebaseServiceMap[]>>(
		'/ds/coderca/service-maps',
	);
	return { data: res.data.data };
};

export default listServiceMaps;
