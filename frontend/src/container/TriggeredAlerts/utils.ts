import { Alerts } from 'types/api/alerts/getTriggered';

import type { Value } from './Filter';

export const FilterAlerts = (
	allAlerts: Alerts[],
	selectedFilter: Value[],
): Alerts[] => {
	// also we need to update the alerts
	// [[key,value]]

	if (selectedFilter?.length === 0 || selectedFilter === undefined) {
		return allAlerts;
	}

	const filter: string[] = [];

	// filtering the value
	selectedFilter.forEach((e) => {
		const valueKey = e.value.split(':');
		if (valueKey.length === 2) {
			filter.push(e.value);
		}
	});

	const tags = filter.map((e) => e.split(':'));
	const objectMap = new Map();

	const filteredKey = tags.reduce((acc, curr) => [...acc, curr[0]], []);
	const filteredValue = tags.reduce((acc, curr) => [...acc, curr[1]], []);

	filteredKey.forEach((key, index) =>
		objectMap.set(key.trim(), filteredValue[index].trim()),
	);

	const filteredAlerts: Set<string> = new Set();

	allAlerts.forEach((alert) => {
		const { labels } = alert;
		if (!labels) {
			return;
		}
		Object.keys(labels).forEach((e) => {
			const selectedKey = objectMap.get(e);

			// alerts which does not have the key with value
			if (selectedKey && labels[e] === selectedKey) {
				filteredAlerts.add(alert.fingerprint);
			}
		});
	});

	return allAlerts.filter((e) => filteredAlerts.has(e.fingerprint));
};

const SEVERITY_RANK: Record<string, number> = {
	critical: 0,
	error: 1,
	warning: 2,
	info: 3,
};
const UNKNOWN_SEVERITY_RANK = 4;

export const severityCompare = (a: Alerts, b: Alerts): number => {
	const aSeverity = (a.labels?.severity || '').toLowerCase();
	const bSeverity = (b.labels?.severity || '').toLowerCase();
	const aRank = SEVERITY_RANK[aSeverity] ?? UNKNOWN_SEVERITY_RANK;
	const bRank = SEVERITY_RANK[bSeverity] ?? UNKNOWN_SEVERITY_RANK;
	if (aRank !== bRank) {
		return aRank - bRank;
	}
	return aSeverity.localeCompare(bSeverity);
};

const STATUS_RANK: Record<string, number> = {
	active: 0,
	suppressed: 1,
	unprocessed: 2,
};
const UNKNOWN_STATUS_RANK = 3;

export const statusCompare = (a: Alerts, b: Alerts): number => {
	const aState = a.status?.state || '';
	const bState = b.status?.state || '';
	const aRank = STATUS_RANK[aState] ?? UNKNOWN_STATUS_RANK;
	const bRank = STATUS_RANK[bState] ?? UNKNOWN_STATUS_RANK;
	if (aRank !== bRank) {
		return aRank - bRank;
	}
	return aState.localeCompare(bState);
};

export const alertNameCompare = (a: Alerts, b: Alerts): number =>
	(a.labels?.alertname || '').localeCompare(b.labels?.alertname || '');

// 백엔드가 트리거된 알림에 붙이는 내부 라벨 — 값은
// pkg/types/ruletypes/labels.go의 AlertRuleIDLabel/RuleSourceLabel과 같아야 한다.
const EXCLUDED_FILTER_OPTION_KEYS = ['ruleId', 'ruleSource'];

export const buildFilterOptions = (
	allAlerts: Alerts[],
): Array<Value & { title: string }> => {
	const uniqueOptions = new Set<string>();

	allAlerts.forEach((alert) => {
		Object.entries(alert.labels || {}).forEach(([key, value]) => {
			if (EXCLUDED_FILTER_OPTION_KEYS.includes(key)) {
				return;
			}
			// FilterAlerts는 split(':') 길이가 정확히 2인 값만 필터로 인정한다 —
			// ':'가 든 키/값(예: ruleSource URL, host:port)은 선택해도 무시되는
			// 함정 옵션이 되므로 옵션에서 제외한다.
			if (key.includes(':') || value.includes(':')) {
				return;
			}
			uniqueOptions.add(`${key}:${value}`);
		});
	});

	return [...uniqueOptions]
		.sort((a, b) => a.localeCompare(b))
		// title: ''는 그룹 셀렉트 옵션과 동일 — 네이티브 title 툴팁을 꺼서
		// TextOverflowTooltip과의 이중 툴팁을 막는다.
		.map((value) => ({ value, title: '' }));
};
