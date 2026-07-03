import {
	NocAlert,
	NocCounts,
	NocSeverity,
	NocServiceRow,
	TrendTarget,
	WatchSelection,
} from '../types';

// §7.1 확정 팔레트 — 슬롯 순서 = 색약 안전 순서(재배열 금지).
export const SERIES_PALETTE_DARK = [
	'#3987e5', // 1 blue
	'#199e70', // 2 aqua
	'#c98500', // 3 yellow
	'#008300', // 4 green
	'#9085e9', // 5 violet
	'#e66767', // 6 red
	'#d55181', // 7 magenta
];

export const SERIES_PALETTE_LIGHT = [
	'#2a78d6',
	'#1baf7a',
	'#eda100',
	'#008300',
	'#4a3aa7',
	'#e34948',
	'#e87ba4',
];

const WATCH_CARD_CAP = 5;
const TREND_CAP = 7;

const SEVERITY_RANK: Record<NocSeverity, number> = {
	critical: 0,
	error: 1,
	warning: 2,
	info: 3,
};

// health -> triage severity ordering (critical worst)
const HEALTH_RANK: Record<NocServiceRow['health'], number> = {
	critical: 0,
	warning: 1,
	healthy: 2,
};

export function deriveCounts(
	services: NocServiceRow[],
	firingCount: number,
): NocCounts {
	let critical = 0;
	let warning = 0;
	let healthy = 0;
	services.forEach((s) => {
		if (s.health === 'critical') {
			critical += 1;
		} else if (s.health === 'warning') {
			warning += 1;
		} else {
			healthy += 1;
		}
	});
	return { critical, warning, healthy, alerts: firingCount };
}

export function selectWatch(services: NocServiceRow[]): WatchSelection {
	const anomalies = services
		.filter((s) => s.health !== 'healthy')
		.sort(
			(a, b) => HEALTH_RANK[a.health] - HEALTH_RANK[b.health] || b.errPct - a.errPct,
		);

	if (anomalies.length > 0) {
		return { services: anomalies.slice(0, WATCH_CARD_CAP), mode: 'anomaly' };
	}

	const watch = [...services]
		.sort((a, b) => b.errPct - a.errPct)
		.slice(0, WATCH_CARD_CAP);
	return { services: watch, mode: 'watch' };
}

export function selectTrendTargets(services: NocServiceRow[]): TrendTarget[] {
	const criticals = services.filter((s) => s.health === 'critical');
	const rest = services
		.filter((s) => s.health !== 'critical')
		.sort(
			(a, b) => HEALTH_RANK[a.health] - HEALTH_RANK[b.health] || b.rps - a.rps,
		);

	// critical은 상한과 무관하게 전부 포함, 남는 슬롯을 rest로 채운다.
	const remainingSlots = Math.max(0, TREND_CAP - criticals.length);
	const chosen = [...criticals, ...rest.slice(0, remainingSlots)];

	// 엔티티에 색 고정: 선택 순서대로 슬롯 배정.
	return chosen.map((s, i) => ({
		name: s.name,
		color: SERIES_PALETTE_DARK[i % SERIES_PALETTE_DARK.length],
	}));
}

export function pickIncident(alerts: NocAlert[]): NocAlert | null {
	const sorted = [...alerts].sort(
		(a, b) => SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity],
	);
	return sorted[0] ?? null;
}
