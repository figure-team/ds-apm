import { renderHook } from '@testing-library/react';
import { ServicesList } from 'types/api/metrics/getService';

import useNocOverview from '../useNocOverview';

// 스펙 §6-1 B안: 완전 mock(동기·결정적). 전역 mock이 없으므로 파일 내 선언 필수.
// getService는 직접 mock 필수(2차 리뷰 C-1): useQuery mock은 "호출"만 차단할 뿐
// "모듈 로드"는 차단하지 못한다. useNocOverview.ts:1의 값 import가
// getService.ts:1 → api/index.ts:21의 top-level `new QueryClient(...)`를 평가시키는데,
// 아래 react-query mock이 QueryClient를 undefined로 만들어 테스트 파일이
// import 단계에서 TypeError로 통째로 죽는다. getService를 직접 mock해 api 모듈
// 그래프 로드 자체를 차단한다(하위 그래프발 2차 크래시 위험도 함께 제거 — 리뷰 M-1).
const mockServices: ServicesList[] = [];

jest.mock('api/metrics/getService', () => ({
	__esModule: true,
	default: jest.fn(),
}));
jest.mock('react-query', () => ({
	useQuery: (): {
		data: ServicesList[];
		isLoading: boolean;
		isError: boolean;
	} => ({ data: mockServices, isLoading: false, isError: false }),
}));
jest.mock('react-redux', () => ({
	useSelector: (selector: (s: unknown) => unknown): unknown =>
		selector({
			globalTime: {
				minTime: 1_700_000_000_000 * 1e6, // ns
				maxTime: 1_700_000_060_000 * 1e6,
				selectedTime: '30m',
			},
		}),
}));
jest.mock('react-i18next', () => ({
	useTranslation: (): { t: (k: string) => string } => ({
		t: (k: string): string => k,
	}),
}));
jest.mock('hooks/useResourceAttribute', () => ({
	__esModule: true,
	default: (): { queries: unknown[] } => ({ queries: [] }),
}));
jest.mock('hooks/useResourceAttribute/utils', () => ({
	convertRawQueriesToTraceSelectedTags: (): unknown[] => [],
}));

function svcItem(
	name: string,
	callRate: number,
	errorRate: number,
	p99: number,
): ServicesList {
	return ({
		serviceName: name,
		callRate,
		errorRate,
		p99,
	} as unknown) as ServicesList;
}

// 20개: healthy 19(RPS 200~20, errorRate 0.2) + RPS 최하위 critical 1(errorRate 12 ≥ 5)
function seed20(): void {
	for (let i = 0; i < 19; i += 1) {
		mockServices.push(svcItem(`svc-${i}`, 200 - i * 10, 0.2, 2e8));
	}
	mockServices.push(svcItem('batch-low', 0.5, 12, 4e8));
}

describe('useNocOverview', () => {
	beforeEach(() => {
		mockServices.length = 0;
	});

	it('returns all 20 rows including the low-RPS critical (no 12-row cut)', () => {
		seed20();
		const { result } = renderHook(() => useNocOverview(0));
		expect(result.current.services).toHaveLength(20);
		const low = result.current.services.find((s) => s.name === 'batch-low');
		expect(low?.health).toBe('critical');
	});

	it('keeps rows sorted by RPS desc', () => {
		seed20();
		const { result } = renderHook(() => useNocOverview(0));
		const rps = result.current.services.map((s) => s.rps);
		expect(rps).toEqual([...rps].sort((a, b) => b - a));
		expect(result.current.services[19].name).toBe('batch-low');
	});

	it('aggregates kpis over the full list (19/20 healthy, uptime 95.0)', () => {
		seed20();
		const { result } = renderHook(() => useNocOverview(0));
		expect(result.current.kpis.find((k) => k.key === 'services')?.value).toBe(
			'19/20',
		);
		expect(result.current.kpis.find((k) => k.key === 'uptime')?.value).toBe(
			'95.0',
		);
	});
});
