import { RemediationTargetWire } from 'api/remediationTargets';

// props 계약은 레인 간 인터페이스로 시드에 고정 — 변경은 오케스트레이터 승인 필요
// (plans/2026-07-02-remtgt-parallel-orchestration.md). 구현은 레인 C2 담당.
export interface TargetFormDrawerProps {
	open: boolean;
	mode: 'create' | 'edit';
	initial?: RemediationTargetWire;
	encryptionReady: boolean;
	onClose: () => void;
	onSaved: () => void;
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function TargetFormDrawer(_props: TargetFormDrawerProps): JSX.Element | null {
	return null;
}

export default TargetFormDrawer;
