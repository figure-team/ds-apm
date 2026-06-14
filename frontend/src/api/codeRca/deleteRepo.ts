import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

const deleteRepo = (repoId: string): Promise<AxiosResponse<void>> =>
	ApiV2Instance.delete<void>(`/ds/coderca/repos/${encodeURIComponent(repoId)}`);

export default deleteRepo;
