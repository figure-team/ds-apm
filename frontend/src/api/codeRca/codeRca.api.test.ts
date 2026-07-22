/**
 * Regression tests for the CF-11 "Something went wrong" crash.
 *
 * Root cause: the backend wraps every render.Success body in
 * `{ status: "success", data: <payload> }`. The read clients used to return
 * `res.data` (the whole envelope object) instead of `res.data.data`, so the
 * ConfigTab handed a non-array object to antd <Table>, whose internals call
 * `dataSource.some(...)` -> "TypeError: B.some is not a function".
 *
 * These tests feed the real envelope shape and assert the clients unwrap it.
 */
import exportRun from './exportRun';
import getConfig from './getConfig';
import getRun from './getRun';
import listRepos from './listRepos';
import listRuns from './listRuns';
import listServiceMaps from './listServiceMaps';

const mockGet = jest.fn();
const mockPost = jest.fn();

jest.mock('api', () => ({
	ApiV2Instance: {
		get: (...args: unknown[]): unknown => mockGet(...args),
		post: (...args: unknown[]): unknown => mockPost(...args),
	},
}));

const envelope = <T>(data: T): { data: { status: string; data: T } } => ({
	data: { status: 'success', data },
});

beforeEach(() => {
	jest.clearAllMocks();
});

describe('codeRca read clients unwrap the render.Success { status, data } envelope', () => {
	it('listRepos returns the inner array, not the envelope object', async () => {
		mockGet.mockResolvedValue(envelope([{ repoId: 'r1' }]));

		const res = await listRepos();

		// Before the fix res.data was { status, data } -> antd Table .some() crash.
		expect(Array.isArray(res.data)).toBe(true);
		expect(res.data).toEqual([{ repoId: 'r1' }]);
	});

	it('listServiceMaps returns the inner array', async () => {
		mockGet.mockResolvedValue(envelope([{ serviceName: 'svc-a' }]));

		const res = await listServiceMaps();

		expect(Array.isArray(res.data)).toBe(true);
		expect(res.data).toEqual([{ serviceName: 'svc-a' }]);
	});

	it('listRuns forwards params and returns the inner array', async () => {
		mockGet.mockResolvedValue(envelope([{ runId: 'run-1' }]));

		const res = await listRuns({ status: 'done' });

		expect(mockGet).toHaveBeenCalledWith('/ds/coderca/runs', {
			params: { status: 'done' },
		});
		expect(Array.isArray(res.data)).toBe(true);
		expect(res.data).toEqual([{ runId: 'run-1' }]);
	});

	it('getConfig returns the inner config object', async () => {
		mockGet.mockResolvedValue(envelope({ enabled: true, minSeverity: 'error' }));

		const res = await getConfig();

		expect(res.data).toEqual({ enabled: true, minSeverity: 'error' });
	});

	it('getRun returns the inner run detail object', async () => {
		mockGet.mockResolvedValue(envelope({ runId: 'run-1', rootCause: 'npe' }));

		const res = await getRun('run-1');

		expect(mockGet).toHaveBeenCalledWith('/ds/coderca/runs/run-1');
		expect(res.data).toEqual({ runId: 'run-1', rootCause: 'npe' });
	});

	it('exportRun posts to the export endpoint and unwraps the path', async () => {
		mockPost.mockResolvedValue(envelope({ path: '/srv/m-project/ds-hub/x.md' }));

		const res = await exportRun('run-1');

		expect(mockPost).toHaveBeenCalledWith('/ds/coderca/runs/run-1/export');
		expect(res.data).toEqual({ path: '/srv/m-project/ds-hub/x.md' });
	});
});
