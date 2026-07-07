import { act, renderHook } from '@testing-library/react';
import { LOCALSTORAGE } from 'constants/localStorage';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { Dashboard } from 'types/api/dashboard/getAll';

import { NocPinnedRef } from '../../types';
import { PIN_CAP } from '../../utils/pinnedPanels';
import useNocPinnedPanels from '../useNocPinnedPanels';

const store = new Map<string, string>();
const getMock = jest.fn(
	(key: string): string | null => store.get(key) ?? null,
);
const setMock = jest.fn(
	(key: string, value: string): boolean => {
		store.set(key, value);
		return true;
	},
);

jest.mock('api/browser/localstorage/get', () => ({
	__esModule: true,
	default: (key: string): string | null => getMock(key),
}));
jest.mock('api/browser/localstorage/set', () => ({
	__esModule: true,
	default: (key: string, value: string): boolean => setMock(key, value),
}));

const dashboards: Dashboard[] = [
	({
		id: 'd1',
		data: {
			title: 'dash-d1',
			version: 'v5',
			variables: {},
			widgets: [
				{
					id: 'w1',
					panelTypes: PANEL_TYPES.TIME_SERIES,
					query: {
						queryType: 'builder',
						promql: [],
						clickhouse_sql: [],
						builder: { queryData: [], queryFormulas: [], queryTraceOperator: [] },
						id: 'q1',
					},
				},
				{
					id: 'w2',
					panelTypes: PANEL_TYPES.BAR,
					query: {
						queryType: 'builder',
						promql: [],
						clickhouse_sql: [],
						builder: { queryData: [], queryFormulas: [], queryTraceOperator: [] },
						id: 'q2',
					},
				},
			],
		},
	} as unknown) as Dashboard,
];

const useGetAllDashboardMock = jest.fn(() => ({
	data: { data: dashboards },
	isLoading: false,
}));

jest.mock('hooks/dashboard/useGetAllDashboard', () => ({
	useGetAllDashboard: (): unknown => useGetAllDashboardMock(),
}));

function seedRefs(refs: NocPinnedRef[]): void {
	store.set(LOCALSTORAGE.NOC_PINNED_PANELS, JSON.stringify(refs));
}

function storedRefs(): NocPinnedRef[] {
	const raw = store.get(LOCALSTORAGE.NOC_PINNED_PANELS);
	return raw ? JSON.parse(raw) : [];
}

describe('useNocPinnedPanels', () => {
	beforeEach(() => {
		store.clear();
		getMock.mockClear();
		setMock.mockClear();
	});

	it('reads persisted refs from localStorage and resolves slots', () => {
		seedRefs([{ dashboardId: 'd1', widgetId: 'w1' }]);
		const { result } = renderHook(() => useNocPinnedPanels());
		expect(result.current.refs).toEqual([
			{ dashboardId: 'd1', widgetId: 'w1' },
		]);
		expect(result.current.slots).toHaveLength(1);
		expect(result.current.slots[0].widget?.id).toBe('w1');
		expect(result.current.slots[0].dashboardTitle).toBe('dash-d1');
		expect(result.current.dashboards).toBe(dashboards);
		expect(result.current.isLoading).toBe(false);
	});

	it('returns empty refs on corrupted or non-array stored value', () => {
		store.set(LOCALSTORAGE.NOC_PINNED_PANELS, '{not json');
		const { result } = renderHook(() => useNocPinnedPanels());
		expect(result.current.refs).toEqual([]);

		store.set(LOCALSTORAGE.NOC_PINNED_PANELS, '{"a":1}');
		const { result: r2 } = renderHook(() => useNocPinnedPanels());
		expect(r2.current.refs).toEqual([]);
	});

	it('drops malformed entries and duplicate widgetIds from storage', () => {
		store.set(
			LOCALSTORAGE.NOC_PINNED_PANELS,
			JSON.stringify([
				{ dashboardId: 'd1', widgetId: 'w1' },
				{ dashboardId: 'd1' }, // widgetId 누락
				{ dashboardId: 'd1', widgetId: 'w1' }, // 중복
				null,
				{ dashboardId: 'd1', widgetId: 'w2' },
			]),
		);
		const { result } = renderHook(() => useNocPinnedPanels());
		expect(result.current.refs).toEqual([
			{ dashboardId: 'd1', widgetId: 'w1' },
			{ dashboardId: 'd1', widgetId: 'w2' },
		]);
	});

	it('pin persists to localStorage (round-trip) and dedups same widgetId', () => {
		const { result } = renderHook(() => useNocPinnedPanels());
		act(() => result.current.pin({ dashboardId: 'd1', widgetId: 'w1' }));
		expect(storedRefs()).toEqual([{ dashboardId: 'd1', widgetId: 'w1' }]);

		// 같은 widgetId 재핀 → 중복 없이 유지
		act(() => result.current.pin({ dashboardId: 'd1', widgetId: 'w1' }));
		expect(result.current.refs).toEqual([
			{ dashboardId: 'd1', widgetId: 'w1' },
		]);
		expect(storedRefs()).toEqual([{ dashboardId: 'd1', widgetId: 'w1' }]);
	});

	it('keeps at most PIN_CAP refs when pinning beyond the cap', () => {
		seedRefs([
			{ dashboardId: 'd1', widgetId: 'a' },
			{ dashboardId: 'd1', widgetId: 'b' },
			{ dashboardId: 'd1', widgetId: 'c' },
			{ dashboardId: 'd1', widgetId: 'd' },
		]);
		const { result } = renderHook(() => useNocPinnedPanels());
		expect(result.current.refs).toHaveLength(PIN_CAP);

		act(() => result.current.pin({ dashboardId: 'd1', widgetId: 'e' }));
		expect(result.current.refs).toHaveLength(PIN_CAP);
		// 뒤에서 PIN_CAP개 유지 — 가장 오래된 a 탈락, e 추가
		expect(result.current.refs.map((r) => r.widgetId)).toEqual([
			'b',
			'c',
			'd',
			'e',
		]);
		expect(storedRefs().map((r) => r.widgetId)).toEqual(['b', 'c', 'd', 'e']);
	});

	it('re-pinning an existing widgetId at cap moves it to the end without eviction', () => {
		seedRefs([
			{ dashboardId: 'd1', widgetId: 'a' },
			{ dashboardId: 'd1', widgetId: 'b' },
			{ dashboardId: 'd1', widgetId: 'c' },
			{ dashboardId: 'd1', widgetId: 'd' },
		]);
		const { result } = renderHook(() => useNocPinnedPanels());
		act(() => result.current.pin({ dashboardId: 'd1', widgetId: 'b' }));
		expect(result.current.refs.map((r) => r.widgetId)).toEqual([
			'a',
			'c',
			'd',
			'b',
		]);
	});

	it('unpin removes the ref and persists', () => {
		seedRefs([
			{ dashboardId: 'd1', widgetId: 'w1' },
			{ dashboardId: 'd1', widgetId: 'w2' },
		]);
		const { result } = renderHook(() => useNocPinnedPanels());
		act(() => result.current.unpin('w1'));
		expect(result.current.refs).toEqual([
			{ dashboardId: 'd1', widgetId: 'w2' },
		]);
		expect(storedRefs()).toEqual([{ dashboardId: 'd1', widgetId: 'w2' }]);
	});

	it('exposes isLoading and empty dashboards while dashboards are loading', () => {
		useGetAllDashboardMock.mockReturnValueOnce(({
			data: undefined,
			isLoading: true,
		} as unknown) as { data: { data: Dashboard[] }; isLoading: boolean });
		seedRefs([{ dashboardId: 'd1', widgetId: 'w1' }]);
		const { result } = renderHook(() => useNocPinnedPanels());
		expect(result.current.isLoading).toBe(true);
		expect(result.current.dashboards).toEqual([]);
		// 대시보드 미도착 시 슬롯은 widget null로 유지(참조는 보존)
		expect(result.current.slots[0].widget).toBeNull();
	});
});
