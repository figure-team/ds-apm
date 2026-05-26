import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';
import { AIConfig } from 'container/AIModuleSettings/AIModuleSettings';

const getAIConfig = (): Promise<AxiosResponse<AIConfig>> =>
	ApiV2Instance.get<AIConfig>('/ds/ai/config');

export default getAIConfig;
