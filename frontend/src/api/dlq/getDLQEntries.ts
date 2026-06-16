import axios from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, SuccessResponseV2 } from 'types/api';
import { DLQEntry, GetDLQEntriesParams } from 'types/api/dlq';

const getDLQEntries = async (
	params: GetDLQEntriesParams = {},
): Promise<SuccessResponseV2<DLQEntry[]>> => {
	try {
		const query: Record<string, string> = {};
		if (params.channel) query.channel = params.channel;
		if (params.status) query.status = params.status;

		const response = await axios.get<{ data: DLQEntry[] }>(
			'/alertmanager/dlq/entries',
			{ params: query },
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

export default getDLQEntries;
