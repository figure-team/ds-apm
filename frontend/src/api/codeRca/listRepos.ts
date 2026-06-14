import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodebaseRepo } from './types';

const listRepos = (): Promise<AxiosResponse<CodebaseRepo[]>> =>
	ApiV2Instance.get<CodebaseRepo[]>('/ds/coderca/repos');

export default listRepos;
