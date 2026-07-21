import { AxiosError } from 'axios';
import { ErrorV2Resp } from 'types/api';
import APIError from 'types/api/error';

import { ErrorResponseHandlerV2 } from '../ErrorResponseHandlerV2';

function makeAxiosError(overrides: Record<string, unknown>): AxiosError<ErrorV2Resp> {
	return {
		name: 'AxiosError',
		message: 'Request failed with status code 502',
		code: 'ERR_BAD_RESPONSE',
		...overrides,
	} as unknown as AxiosError<ErrorV2Resp>;
}

describe('ErrorResponseHandlerV2', () => {
	it('정상 엔벨로프 응답은 code/message를 그대로 옮긴다', () => {
		const err = makeAxiosError({
			response: {
				status: 403,
				data: {
					error: { code: 'forbidden', message: '권한 없음', url: '', errors: [] },
				},
			},
		});
		try {
			ErrorResponseHandlerV2(err);
			fail('throw 되어야 한다');
		} catch (e) {
			expect(e).toBeInstanceOf(APIError);
			expect((e as APIError).getHttpStatusCode()).toBe(403);
			expect((e as APIError).getErrorCode()).toBe('forbidden');
			expect((e as APIError).getErrorMessage()).toBe('권한 없음');
		}
	});

	it('엔벨로프가 아닌 5xx(HTML 본문)도 APIError로 정규화한다', () => {
		const err = makeAxiosError({
			response: { status: 502, data: '<html>Bad Gateway</html>' },
		});
		try {
			ErrorResponseHandlerV2(err);
			fail('throw 되어야 한다');
		} catch (e) {
			expect(e).toBeInstanceOf(APIError);
			expect((e as APIError).getHttpStatusCode()).toBe(502);
			expect((e as APIError).getErrorCode()).toBe('ERR_BAD_RESPONSE');
			expect((e as APIError).getErrorMessage()).toBe(
				'Request failed with status code 502',
			);
		}
	});

	it('data가 아예 없는 응답도 APIError로 정규화한다', () => {
		const err = makeAxiosError({ response: { status: 500, data: undefined } });
		try {
			ErrorResponseHandlerV2(err);
			fail('throw 되어야 한다');
		} catch (e) {
			expect(e).toBeInstanceOf(APIError);
			expect((e as APIError).getHttpStatusCode()).toBe(500);
		}
	});

	it('응답이 없고 요청만 있는 경우도 APIError를 던진다', () => {
		const err = makeAxiosError({ request: {}, status: undefined });
		try {
			ErrorResponseHandlerV2(err);
			fail('throw 되어야 한다');
		} catch (e) {
			expect(e).toBeInstanceOf(APIError);
			expect((e as APIError).getHttpStatusCode()).toBe(500);
		}
	});
});
