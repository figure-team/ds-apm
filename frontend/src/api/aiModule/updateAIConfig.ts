import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';
import { AIConfigPayload } from 'container/AIModuleSettings/AIModuleSettings';

const updateAIConfig = (
	body: AIConfigPayload,
): Promise<AxiosResponse<AIConfigPayload>> =>
	ApiV2Instance.put<AIConfigPayload>('/ds/ai/config', body);

export default updateAIConfig;
