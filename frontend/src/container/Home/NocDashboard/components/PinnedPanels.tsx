import { NocPinnedSlot } from '../types';

export interface PinnedPanelsProps {
	slots: NocPinnedSlot[];
	onUnpin: (widgetId: string) => void;
	onOpenPicker: () => void;
}

// Lane C1 소유 — 시드 스텁 (impl-plan Task 5).
// GridCardGraph 임베드 슬롯 + "패널 고정" 추가 타일은 Lane C1이 구현.
export default function PinnedPanels({
	slots,
}: PinnedPanelsProps): JSX.Element {
	return <div className="noc-c2-pins" data-slot-count={slots.length} />;
}
