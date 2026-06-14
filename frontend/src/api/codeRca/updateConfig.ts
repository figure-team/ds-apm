import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodeRcaConfig } from './types';

const updateConfig = (body: CodeRcaConfig): Promise<AxiosResponse<void>> =>
	ApiV2Instance.put<void>('/ds/coderca/config', body);

export default updateConfig;
