import type { Value } from '../Filter';
import {
	alertNameCompare,
	buildFilterOptions,
	FilterAlerts,
	severityCompare,
	statusCompare,
} from '../utils';
import { createAlert } from './mockUtils';

describe('FilterAlerts', () => {
	it('returns all alerts when no filters are selected', () => {
		const alerts = [
			createAlert({ fingerprint: 'fp-1' }),
			createAlert({ fingerprint: 'fp-2' }),
		];
		const filters: Value[] = [];

		const result = FilterAlerts(alerts, filters);

		expect(result).toBe(alerts);
	});

	it('filters alerts that have matching label key and value', () => {
		const warningAlert = createAlert({
			fingerprint: 'warning',
			labels: { severity: 'warning' },
		});
		const criticalAlert = createAlert({
			fingerprint: 'critical',
			labels: { severity: 'critical' },
		});
		const alerts = [warningAlert, criticalAlert];
		const filters: Value[] = [{ value: 'severity:critical' }];

		const result = FilterAlerts(alerts, filters);

		expect(result).toEqual([criticalAlert]);
	});

	it('includes alerts when any filter matches', () => {
		const severityAlert = createAlert({
			fingerprint: 'severity',
			labels: { severity: 'warning' },
		});
		const teamAlert = createAlert({
			fingerprint: 'team',
			labels: { team: 'core-observability' },
		});
		const otherAlert = createAlert({
			fingerprint: 'other',
			labels: { service: 'ingestor' },
		});
		const alerts = [severityAlert, teamAlert, otherAlert];
		const filters: Value[] = [
			{ value: 'severity:warning' },
			{ value: 'team:core-observability' },
		];

		const result = FilterAlerts(alerts, filters);

		expect(result).toHaveLength(2);
		expect(result).toEqual([severityAlert, teamAlert]);
	});

	it('matches labels even when filters contain surrounding whitespace', () => {
		const alert = createAlert({
			fingerprint: 'trim-test',
			labels: { severity: 'critical' },
		});
		const alerts = [alert];
		const filters: Value[] = [{ value: '  severity  :  critical  ' }];

		const result = FilterAlerts(alerts, filters);

		expect(result).toEqual([alert]);
	});

	it('ignores filters that do not contain a key/value delimiter', () => {
		const alert = createAlert({
			fingerprint: 'invalid-filter',
			labels: { severity: 'warning' },
		});
		const alerts = [alert];
		const filters: Value[] = [{ value: 'severitywarning' }];

		const result = FilterAlerts(alerts, filters);

		expect(result).toEqual([]);
	});
});

describe('severityCompare', () => {
	it('orders known severities critical > error > warning > info ascending', () => {
		const critical = createAlert({ labels: { severity: 'critical' } });
		const error = createAlert({ labels: { severity: 'error' } });
		const warning = createAlert({ labels: { severity: 'warning' } });
		const info = createAlert({ labels: { severity: 'info' } });

		const sorted = [info, error, critical, warning].sort(severityCompare);

		expect(sorted.map((a) => a.labels?.severity)).toEqual([
			'critical',
			'error',
			'warning',
			'info',
		]);
	});

	it('normalizes case before ranking', () => {
		const upper = createAlert({ labels: { severity: 'CRITICAL' } });
		const lower = createAlert({ labels: { severity: 'warning' } });

		expect(severityCompare(upper, lower)).toBeLessThan(0);
	});

	it('ranks unknown severities after known ones with alphabetical tie-break', () => {
		const p1 = createAlert({ labels: { severity: 'P1' } });
		const p2 = createAlert({ labels: { severity: 'P2' } });
		const info = createAlert({ labels: { severity: 'info' } });

		const sorted = [p2, p1, info].sort(severityCompare);

		expect(sorted.map((a) => a.labels?.severity)).toEqual(['info', 'P1', 'P2']);
	});

	it('treats missing severity as unknown rank', () => {
		const none = createAlert({ labels: { alertname: 'no-severity' } });
		const info = createAlert({ labels: { severity: 'info' } });

		expect(severityCompare(info, none)).toBeLessThan(0);
	});
});

describe('statusCompare', () => {
	const withState = (state: string): ReturnType<typeof createAlert> =>
		createAlert({ status: { inhibitedBy: [], silencedBy: [], state } });

	it('orders active before suppressed before unprocessed', () => {
		const sorted = [
			withState('unprocessed'),
			withState('active'),
			withState('suppressed'),
		].sort(statusCompare);

		expect(sorted.map((a) => a.status.state)).toEqual([
			'active',
			'suppressed',
			'unprocessed',
		]);
	});

	it('ranks unknown states last', () => {
		expect(statusCompare(withState('unprocessed'), withState('zombie'))).toBeLessThan(0);
	});
});

describe('alertNameCompare', () => {
	it('compares the full alert name, not just the first character', () => {
		const beta = createAlert({ labels: { alertname: 'Apple Beta' } });
		const alpha = createAlert({ labels: { alertname: 'Apple Alpha' } });

		expect(alertNameCompare(alpha, beta)).toBeLessThan(0);
	});

	it('treats missing names as empty strings', () => {
		const unnamed = createAlert({ labels: {} });
		const named = createAlert({ labels: { alertname: 'A' } });

		expect(alertNameCompare(unnamed, named)).toBeLessThan(0);
	});
});

describe('buildFilterOptions', () => {
	it('builds unique sorted key:value options from alert labels', () => {
		const alerts = [
			createAlert({ labels: { severity: 'warning', team: 'core' } }),
			createAlert({ labels: { severity: 'warning', service: 'api' } }),
		];

		expect(buildFilterOptions(alerts)).toEqual([
			{ value: 'service:api', title: '' },
			{ value: 'severity:warning', title: '' },
			{ value: 'team:core', title: '' },
		]);
	});

	it('excludes internal labels and values containing a colon', () => {
		const alerts = [
			createAlert({
				labels: {
					severity: 'critical',
					ruleId: 'rule-uuid-1',
					ruleSource: 'http://host:8080/alerts/edit',
					endpoint: 'host:443',
				},
			}),
		];

		expect(buildFilterOptions(alerts)).toEqual([
			{ value: 'severity:critical', title: '' },
		]);
	});

	it('returns an empty array when alerts have no labels', () => {
		expect(buildFilterOptions([createAlert()])).toEqual([]);
	});
});
