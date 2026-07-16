import { NocAlert, NocServiceRow } from '../../types';
import {
	deriveCounts,
	pickIncident,
	selectTrendTargets,
	selectWatch,
	SERIES_PALETTE_DARK,
} from '../deriveState';

function svc(over: Partial<NocServiceRow>): NocServiceRow {
	return { name: 'x', health: 'healthy', p99Ms: 10, errPct: 0, rps: 5, ...over };
}

describe('deriveCounts', () => {
	it('counts by health and passes through firing alerts', () => {
		const services = [
			svc({ name: 'a', health: 'critical' }),
			svc({ name: 'b', health: 'warning' }),
			svc({ name: 'c', health: 'healthy' }),
			svc({ name: 'd', health: 'healthy' }),
		];
		expect(deriveCounts(services, 3)).toEqual({
			critical: 1,
			warning: 1,
			healthy: 2,
			alerts: 3,
		});
	});
});

describe('selectWatch', () => {
	it('anomaly mode: critical+warning sorted by severity, cap 5', () => {
		const services = [
			svc({ name: 'ok', health: 'healthy', errPct: 0.2 }),
			svc({ name: 'w', health: 'warning', errPct: 2 }),
			svc({ name: 'c', health: 'critical', errPct: 9 }),
		];
		const r = selectWatch(services);
		expect(r.mode).toBe('anomaly');
		expect(r.services.map((s) => s.name)).toEqual(['c', 'w']);
	});

	it('watch mode: all healthy -> top5 by errPct desc', () => {
		const services = Array.from({ length: 7 }, (_, i) =>
			svc({ name: `s${i}`, health: 'healthy', errPct: i * 0.1 }),
		);
		const r = selectWatch(services);
		expect(r.mode).toBe('watch');
		expect(r.services).toHaveLength(5);
		expect(r.services[0].name).toBe('s6'); // highest errPct first
	});
});

describe('selectTrendTargets', () => {
	it('all critical are included even beyond cap 7 (pushes out healthy)', () => {
		const criticals = Array.from({ length: 8 }, (_, i) =>
			svc({ name: `c${i}`, health: 'critical', rps: 1 }),
		);
		const healthy = svc({ name: 'h', health: 'healthy', rps: 999 });
		const r = selectTrendTargets([healthy, ...criticals]);
		expect(r).toHaveLength(8); // 8 criticals — cap of 7 overridden to keep all critical
		expect(r.every((t) => t.name.startsWith('c'))).toBe(true);
		expect(r.map((t) => t.name)).not.toContain('h');
	});

	it('assigns palette slots in order and fixes color to entity', () => {
		const services = [
			svc({ name: 'a', health: 'critical', rps: 10 }),
			svc({ name: 'b', health: 'warning', rps: 8 }),
		];
		const r = selectTrendTargets(services);
		expect(r[0]).toEqual({ name: 'a', color: SERIES_PALETTE_DARK[0] });
		expect(r[1]).toEqual({ name: 'b', color: SERIES_PALETTE_DARK[1] });
	});

	it('caps at 7 when no criticals force overflow', () => {
		const services = Array.from({ length: 10 }, (_, i) =>
			svc({ name: `s${i}`, health: 'healthy', rps: 10 - i }),
		);
		expect(selectTrendTargets(services)).toHaveLength(7);
	});

	it('hard-caps at 12 keeping highest-RPS criticals (input is RPS-desc)', () => {
		const criticals = Array.from({ length: 15 }, (_, i) =>
			svc({ name: `c${i}`, health: 'critical', rps: 15 - i }),
		);
		const r = selectTrendTargets(criticals);
		expect(r).toHaveLength(12);
		expect(r.map((t) => t.name)).toEqual(
			Array.from({ length: 12 }, (_, i) => `c${i}`),
		);
	});
});

describe('pickIncident', () => {
	it('returns highest-severity firing alert or null', () => {
		const alerts: NocAlert[] = [
			{ id: '1', severity: 'warning', title: 'w', meta: '', age: '', state: 'firing' },
			{ id: '2', severity: 'critical', title: 'c', meta: '', age: '', state: 'firing' },
		];
		expect(pickIncident(alerts)?.id).toBe('2');
		expect(pickIncident([])).toBeNull();
	});

	it('ignores non-firing rules even if severity is higher', () => {
		const alerts: NocAlert[] = [
			{ id: '1', severity: 'critical', title: 'c', meta: '', age: '', state: 'inactive' },
			{ id: '2', severity: 'warning', title: 'w', meta: '', age: '', state: 'firing' },
		];
		expect(pickIncident(alerts)?.id).toBe('2');
	});

	it('returns null when nothing is firing', () => {
		const alerts: NocAlert[] = [
			{ id: '1', severity: 'critical', title: 'c', meta: '', age: '15d', state: 'inactive' },
			{ id: '2', severity: 'error', title: 'e', meta: '', age: '', state: 'disabled' },
		];
		expect(pickIncident(alerts)).toBeNull();
	});
});
