/**
 * 일괄 등록 API의 에러 메시지를 뽑는다.
 *
 * `utils/errorUtils.ts`의 `getApiErrorMessage`와 합치면 안 된다 — 그쪽은 생성 API 형상인
 * `response.data.error.message`를 읽고, 여기는 v2 rules/runbook 형상인
 * `response.data`(문자열) 또는 `response.data.error | response.data.message`를 읽는다.
 * 읽는 위치가 달라서 통합하면 메시지가 조용히 fallback으로 떨어진다.
 */
export function getBulkErrorMessage(error: unknown): string {
	if (typeof error === 'object' && error !== null && 'response' in error) {
		const { response } = error as {
			response?: { data?: { error?: string; message?: string } | string };
		};
		if (typeof response?.data === 'string') {
			return response.data;
		}
		return response?.data?.error || response?.data?.message || '요청 실패';
	}
	return error instanceof Error ? error.message : '요청 실패';
}
