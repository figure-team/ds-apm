import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';
import {
	AIConfigPayload,
	AIConfigTestResult,
} from 'container/AIModuleSettings/AIModuleSettings';

const testAIConfig = (
	body: AIConfigPayload,
): Promise<AxiosResponse<AIConfigTestResult>> =>
	ApiV2Instance.post<AIConfigTestResult>('/ds/ai/config/test', body);

export default testAIConfig;
