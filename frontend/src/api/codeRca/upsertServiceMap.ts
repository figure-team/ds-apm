import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodebaseServiceMap } from './types';

const upsertServiceMap = (
	body: CodebaseServiceMap,
): Promise<AxiosResponse<void>> =>
	ApiV2Instance.put<void>('/ds/coderca/service-maps', body);

export default upsertServiceMap;
