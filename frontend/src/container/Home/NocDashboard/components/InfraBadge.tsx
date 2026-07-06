import { InfraPanelProps } from './InfraPanel';

// Lane A 소유 — 시드 스텁 (impl-plan Task 2).
// antd Popover(placement bottomRight) + InfraPanel 재사용 말풍선은 Lane A가 구현.
export default function InfraBadge({ hosts }: InfraPanelProps): JSX.Element {
	return (
		<button type="button" className="noc-c2-infra-badge noc-c2-infra-calm">
			{hosts.length}
		</button>
	);
}
