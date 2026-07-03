export interface OkStripProps {
	names: string[];
	maxChips?: number;
}

// SEED STUB — 본문은 Lane C가 채움(impl-plan Task 8). props 인터페이스는 계약(불변).
export default function OkStrip(_props: OkStripProps): JSX.Element {
	return <div className="noc-ok-strip noc-c2-okstrip" data-stub="okstrip" />;
}
