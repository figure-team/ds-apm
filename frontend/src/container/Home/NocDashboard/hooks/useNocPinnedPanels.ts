import { Dashboard } from 'types/api/dashboard/getAll';

import { NocPinnedRef, NocPinnedSlot } from '../types';

export interface UseNocPinnedPanelsResult {
	slots: NocPinnedSlot[];
	refs: NocPinnedRef[];
	dashboards: Dashboard[];
	pin: (ref: NocPinnedRef) => void;
	unpin: (widgetId: string) => void;
	isLoading: boolean;
}

// Lane B 소유 — 시드 스텁 (impl-plan Task 4).
// localStorage(NOC_PINNED_PANELS) 영속 + useGetAllDashboard 해석은 Lane B가 구현.
export default function useNocPinnedPanels(): UseNocPinnedPanelsResult {
	return {
		slots: [],
		refs: [],
		dashboards: [],
		pin: (): void => undefined,
		unpin: (): void => undefined,
		isLoading: false,
	};
}
