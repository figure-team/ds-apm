import axios from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, SuccessResponseV2 } from 'types/api';
import { ReplayDLQEntriesPayload, ReplayResult } from 'types/api/dlq';

const replayDLQEntries = async (
	payload: ReplayDLQEntriesPayload,
): Promise<SuccessResponseV2<ReplayResult>> => {
	try {
		const response = await axios.post<{ data: ReplayResult }>(
			'/alertmanager/dlq/replay',
			payload,
		);
		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		ErrorResponseHandlerV2(error as AxiosError<ErrorV2Resp>);
		throw error;
	}
};

export default replayDLQEntries;
