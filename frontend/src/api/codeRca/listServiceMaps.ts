import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodebaseServiceMap } from './types';

const listServiceMaps = (): Promise<AxiosResponse<CodebaseServiceMap[]>> =>
	ApiV2Instance.get<CodebaseServiceMap[]>('/ds/coderca/service-maps');

export default listServiceMaps;
