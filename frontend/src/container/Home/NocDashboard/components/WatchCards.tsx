import { NocServiceRow } from '../types';

export interface WatchCardsProps {
	services: NocServiceRow[];
	mode: 'anomaly' | 'watch';
	overflowCount?: number;
}

// SEED STUB — 본문은 Lane C가 채움(impl-plan Task 6). props 인터페이스는 계약(불변).
export default function WatchCards(_props: WatchCardsProps): JSX.Element {
	return <div className="noc-c2-watch" data-stub="watchcards" />;
}
