import { Dashboard } from 'types/api/dashboard/getAll';

import { NocPinnedRef } from '../types';

export interface PinPickerDrawerProps {
	open: boolean;
	onClose: () => void;
	dashboards: Dashboard[];
	refs: NocPinnedRef[];
	onPin: (ref: NocPinnedRef) => void;
	onUnpin: (widgetId: string) => void;
}

// Lane C2 소유 — 시드 스텁 (impl-plan Task 6).
// antd Drawer + 대시보드별 Collapse + 핀 체크박스는 Lane C2가 구현.
export default function PinPickerDrawer({
	open,
}: PinPickerDrawerProps): JSX.Element | null {
	return open ? <div className="noc-c2-pin-drawer" /> : null;
}
