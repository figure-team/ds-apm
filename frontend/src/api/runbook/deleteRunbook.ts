import { GeneratedAPIInstance } from 'api/generatedAPIInstance';

type ApiResponse<T> = {
	data: T;
	status: string;
};

const deleteRunbook = (
	sopId: string,
	version: string,
	runbookId: string,
): Promise<ApiResponse<void>> => {
	return GeneratedAPIInstance<ApiResponse<void>>({
		url: `/api/v2/ds/sop/documents/${encodeURIComponent(sopId)}/versions/${encodeURIComponent(version)}/runbooks/${encodeURIComponent(runbookId)}`,
		method: 'DELETE',
	});
};

export default deleteRunbook;
