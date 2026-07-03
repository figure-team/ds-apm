import { NocAlert } from '../types';

export interface AlertsPanelProps {
	alerts: NocAlert[];
	isLoading: boolean;
	isError: boolean;
	lastResolved?: { age: string; service: string };
}

// SEED STUB — 본문은 Lane D가 채움(impl-plan Task 9). props 인터페이스는 계약(불변).
export default function AlertsPanel(_props: AlertsPanelProps): JSX.Element {
	return <div className="noc-c2-alerts" data-stub="alertspanel" />;
}
