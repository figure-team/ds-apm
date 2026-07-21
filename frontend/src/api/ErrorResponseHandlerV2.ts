import { AxiosError } from 'axios';
import { ErrorV2Resp } from 'types/api';
import APIError from 'types/api/error';

// reference - https://axios-http.com/docs/handling_errors
export function ErrorResponseHandlerV2(error: AxiosError<ErrorV2Resp>): never {
	const { response, request } = error;
	// The request was made and the server responded with a status code
	// that falls out of the range of 2xx
	if (response) {
		// 게이트웨이 5xx·프록시 에러 등 엔벨로프({error:{...}})가 아닌 응답이 오면
		// data.error가 없다 — 그대로 접근하면 raw TypeError가 새어 나가 앱 전체가
		// 크래시하므로 항상 APIError로 정규화한다.
		const respError = response.data?.error;
		throw new APIError({
			httpStatusCode: response.status || 500,
			error: {
				code: respError?.code ?? error.code ?? error.name,
				message: respError?.message ?? error.message,
				url: respError?.url ?? '',
				errors: respError?.errors ?? [],
			},
		});
	}
	// The request was made but no response was received
	// `error.request` is an instance of XMLHttpRequest in the browser and an instance of
	// http.ClientRequest in node.js
	if (request) {
		throw new APIError({
			httpStatusCode: error.status || 500,
			error: {
				code: error.code || error.name,
				message: error.message,
				url: '',
				errors: [],
			},
		});
	}

	// Something happened in setting up the request that triggered an Error
	throw new APIError({
		httpStatusCode: error.status || 500,
		error: {
			code: error.name,
			message: error.message,
			url: '',
			errors: [],
		},
	});
}
