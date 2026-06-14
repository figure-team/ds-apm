import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodeRcaConfig } from './types';

const getConfig = (): Promise<AxiosResponse<CodeRcaConfig>> =>
	ApiV2Instance.get<CodeRcaConfig>('/ds/coderca/config');

export default getConfig;
