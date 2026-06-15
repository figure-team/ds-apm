import { ApiV2Instance } from 'api';
import { AIConfig } from 'container/AIModuleSettings/AIModuleSettings';

// Backend wraps render.Success responses in { status, data }; unwrap one level
// so the consumer receives the bare AIConfig (otherwise saved values never load).
interface ApiEnvelope<T> {
	status: string;
	data: T;
}

const getAIConfig = async (): Promise<{ data: AIConfig }> => {
	const res = await ApiV2Instance.get<ApiEnvelope<AIConfig>>('/ds/ai/config');
	return { data: res.data.data };
};

export default getAIConfig;
