import { ApiV2Instance } from 'api';

import { ApiEnvelope, CodebaseRepo } from './types';

const listRepos = async (): Promise<{ data: CodebaseRepo[] }> => {
	const res = await ApiV2Instance.get<ApiEnvelope<CodebaseRepo[]>>(
		'/ds/coderca/repos',
	);
	return { data: res.data.data };
};

export default listRepos;
