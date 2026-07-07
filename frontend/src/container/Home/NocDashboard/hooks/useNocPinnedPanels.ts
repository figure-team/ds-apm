import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import { LOCALSTORAGE } from 'constants/localStorage';
import { useGetAllDashboard } from 'hooks/dashboard/useGetAllDashboard';
import { useCallback, useMemo, useState } from 'react';
import { Dashboard } from 'types/api/dashboard/getAll';

import { NocPinnedRef, NocPinnedSlot } from '../types';
import { PIN_CAP, resolvePinnedSlots } from '../utils/pinnedPanels';

function readRefs(): NocPinnedRef[] {
	try {
		const raw = getLocalStorageApi(LOCALSTORAGE.NOC_PINNED_PANELS);
		const parsed = raw ? JSON.parse(raw) : [];
		if (!Array.isArray(parsed)) return [];
		const seen = new Set<string>();
		return parsed
			.filter(
				(r): r is NocPinnedRef =>
					typeof r?.dashboardId === 'string' && typeof r?.widgetId === 'string',
			)
			.filter((r) => {
				// 손상된 저장값의 중복 widgetId 방어 — PinnedPanels가 widgetId를 React key로 씀
				if (seen.has(r.widgetId)) return false;
				seen.add(r.widgetId);
				return true;
			})
			.slice(0, PIN_CAP);
	} catch {
		return [];
	}
}

export interface UseNocPinnedPanelsResult {
	slots: NocPinnedSlot[];
	refs: NocPinnedRef[];
	dashboards: Dashboard[];
	pin: (ref: NocPinnedRef) => void;
	unpin: (widgetId: string) => void;
	isLoading: boolean;
}

// Lane B — localStorage(NOC_PINNED_PANELS) 영속 + useGetAllDashboard 해석 (impl-plan Task 4)
export default function useNocPinnedPanels(): UseNocPinnedPanelsResult {
	const { data, isLoading } = useGetAllDashboard();
	const [refs, setRefs] = useState<NocPinnedRef[]>(readRefs);

	const persist = useCallback((next: NocPinnedRef[]): NocPinnedRef[] => {
		setLocalStorageApi(LOCALSTORAGE.NOC_PINNED_PANELS, JSON.stringify(next));
		return next;
	}, []);

	const pin = useCallback(
		(ref: NocPinnedRef): void => {
			setRefs((prev) => {
				// 동일 widgetId 중복 제거 후 뒤에 추가, 뒤에서 PIN_CAP개 유지
				const without = prev.filter((r) => r.widgetId !== ref.widgetId);
				return persist([...without, ref].slice(-PIN_CAP));
			});
		},
		[persist],
	);

	const unpin = useCallback(
		(widgetId: string): void => {
			setRefs((prev) => persist(prev.filter((r) => r.widgetId !== widgetId)));
		},
		[persist],
	);

	const dashboards = useMemo(() => data?.data ?? [], [data]);
	const slots = useMemo(() => resolvePinnedSlots(dashboards, refs), [
		dashboards,
		refs,
	]);

	return { slots, refs, dashboards, pin, unpin, isLoading };
}
