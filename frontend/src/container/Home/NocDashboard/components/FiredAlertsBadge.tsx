export interface FiredAlertsBadgeProps {
	count: number;
}

// Lane A 소유 — 시드 스텁 (impl-plan Task 1).
// 클릭 → ROUTES.LIST_ALL_ALERT 이동, count 0이면 noc-c2-fired-quiet 톤은 Lane A가 구현.
export default function FiredAlertsBadge({
	count,
}: FiredAlertsBadgeProps): JSX.Element {
	return (
		<button type="button" className="noc-c2-fired noc-c2-fired-quiet">
			{count}
		</button>
	);
}
