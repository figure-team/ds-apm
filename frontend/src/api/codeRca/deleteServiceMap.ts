import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

const deleteServiceMap = (
	serviceName: string,
): Promise<AxiosResponse<void>> =>
	ApiV2Instance.delete<void>(
		`/ds/coderca/service-maps/${encodeURIComponent(serviceName)}`,
	);

export default deleteServiceMap;
