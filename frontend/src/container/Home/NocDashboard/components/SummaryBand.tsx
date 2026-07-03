import { NocAlert, NocCounts } from '../types';

export interface SummaryBandProps {
	counts: NocCounts;
	incident: NocAlert | null;
	stableSince?: string;
}

// SEED STUB — 본문은 Lane C가 채움(impl-plan Task 5). props 인터페이스는 계약(불변).
export default function SummaryBand(_props: SummaryBandProps): JSX.Element {
	return <div className="noc-c2-band" data-stub="summaryband" />;
}
