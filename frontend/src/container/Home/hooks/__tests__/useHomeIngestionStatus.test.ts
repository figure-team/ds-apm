import { renderHook, waitFor } from '@testing-library/react';

import useHomeIngestionStatus from '../useHomeIngestionStatus';

const mockUseGetQueryRange = jest.fn();
const mockUseGetMetricsOnboardingStatus = jest.fn();
const mockIsIngestionActive = jest.fn();

jest.mock('hooks/queryBuilder/useGetQueryRange', () => ({
	useGetQueryRange: (...args: unknown[]): unknown =>
		mockUseGetQueryRange(...args),
}));

jest.mock('api/generated/services/metrics', () => ({
	useGetMetricsOnboardingStatus: (): unknown =>
		mockUseGetMetricsOnboardingStatus(),
}));

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: jest.fn(),
}));

jest.mock('utils/app', () => ({
	...jest.requireActual('utils/app'),
	isIngestionActive: (payload: unknown): boolean =>
		mockIsIngestionActive(payload),
}));

describe('useHomeIngestionStatus', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockUseGetQueryRange.mockReturnValue({ data: undefined, isLoading: false });
		mockUseGetMetricsOnboardingStatus.mockReturnValue({ data: undefined });
		mockIsIngestionActive.mockReturnValue(false);
	});

	it('reports nothing active when no telemetry has arrived', async () => {
		const { result } = renderHook(() => useHomeIngestionStatus());
		await waitFor(() => {
			expect(result.current.isAnyIngestionActive).toBe(false);
		});
		expect(result.current.showNocDashboard).toBe(false);
	});

	it('flags logs and traces active when the payload reports ingestion', async () => {
		mockUseGetQueryRange.mockReturnValue({
			data: { payload: { some: 'data' } },
			isLoading: false,
		});
		mockIsIngestionActive.mockReturnValue(true);

		const { result } = renderHook(() => useHomeIngestionStatus());

		await waitFor(() => {
			expect(result.current.isLogsIngestionActive).toBe(true);
		});
		expect(result.current.isTracesIngestionActive).toBe(true);
		expect(result.current.isAnyIngestionActive).toBe(true);
		expect(result.current.showNocDashboard).toBe(true);
	});

	it('flags metrics active from the onboarding status', async () => {
		mockUseGetMetricsOnboardingStatus.mockReturnValue({
			data: { data: { hasMetrics: true } },
		});

		const { result } = renderHook(() => useHomeIngestionStatus());

		await waitFor(() => {
			expect(result.current.isMetricsIngestionActive).toBe(true);
		});
		expect(result.current.isAnyIngestionActive).toBe(true);
	});

	it('latches an active flag on — it never flips back to false', async () => {
		mockUseGetQueryRange.mockReturnValue({
			data: { payload: { some: 'data' } },
			isLoading: false,
		});
		mockIsIngestionActive.mockReturnValue(true);

		const { result, rerender } = renderHook(() => useHomeIngestionStatus());
		await waitFor(() => {
			expect(result.current.isLogsIngestionActive).toBe(true);
		});

		// 반드시 **새 data 객체**를 돌려줘야 한다. 래치 effect의 deps가 `logsData`
		// 참조이므로 같은 참조로 rerender하면 effect가 아예 재실행되지 않아
		// 래치를 제거해도 이 테스트가 통과한다(= 무의미한 테스트).
		mockUseGetQueryRange.mockReturnValue({
			data: { payload: { some: 'other data' } },
			isLoading: false,
		});
		mockIsIngestionActive.mockReturnValue(false);
		rerender();

		await waitFor(() => {
			expect(mockIsIngestionActive).toHaveBeenCalled();
		});
		expect(result.current.isLogsIngestionActive).toBe(true);
	});

	it('surfaces the loading flags from the range queries', () => {
		mockUseGetQueryRange.mockReturnValue({ data: undefined, isLoading: true });
		const { result } = renderHook(() => useHomeIngestionStatus());
		expect(result.current.isLogsLoading).toBe(true);
		expect(result.current.isTracesLoading).toBe(true);
	});
});
