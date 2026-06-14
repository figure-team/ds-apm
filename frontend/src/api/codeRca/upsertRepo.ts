import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodebaseRepo } from './types';

const upsertRepo = (body: CodebaseRepo): Promise<AxiosResponse<void>> =>
	ApiV2Instance.put<void>('/ds/coderca/repos', body);

export default upsertRepo;
