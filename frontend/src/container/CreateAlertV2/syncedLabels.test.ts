import {
	extractManagedLabels,
	isManagedLabelKey,
	mergeManagedLabels,
	syncLabelsToExpression,
} from './syncedLabels';

describe('syncedLabels - extractManagedLabels (query -> labels)', () => {
	it('extracts a single equality service.name filter', () => {
		expect(extractManagedLabels(`service.name = 'frontend'`)).toEqual({
			'service.name': 'frontend',
		});
	});

	it('extracts multiple managed attributes', () => {
		expect(
			extractManagedLabels(
				`service.name = 'frontend' AND deployment.environment = 'prod'`,
			),
		).toEqual({
			'service.name': 'frontend',
			deployment_environment: 'prod',
		});
	});

	it('extracts a single-valued IN filter', () => {
		expect(extractManagedLabels(`service.name in ['frontend']`)).toEqual({
			'service.name': 'frontend',
		});
	});

	it('ignores multi-valued IN filters', () => {
		expect(
			extractManagedLabels(`service.name in ['frontend', 'backend']`),
		).toEqual({});
	});

	it('ignores negation and regex operators', () => {
		expect(extractManagedLabels(`service.name != 'frontend'`)).toEqual({});
		expect(extractManagedLabels(`service.name regex 'front.*'`)).toEqual({});
	});

	it('ignores non-managed attribute keys', () => {
		expect(extractManagedLabels(`http.status_code = '500'`)).toEqual({});
	});

	it('keeps managed keys while ignoring non-managed ones in the same expression', () => {
		expect(
			extractManagedLabels(
				`service.name = 'frontend' AND http.status_code = '500'`,
			),
		).toEqual({ 'service.name': 'frontend' });
	});

	it('returns empty object for empty expression', () => {
		expect(extractManagedLabels('')).toEqual({});
	});
});

describe('syncedLabels - mergeManagedLabels', () => {
	it('adds managed labels from the query while preserving severity and custom labels', () => {
		expect(
			mergeManagedLabels(
				{ severity: 'critical', team: 'payments' },
				{ 'service.name': 'frontend' },
			),
		).toEqual({
			severity: 'critical',
			team: 'payments',
			'service.name': 'frontend',
		});
	});

	it('removes managed labels that are no longer in the query', () => {
		expect(
			mergeManagedLabels({ severity: 'warning', 'service.name': 'frontend' }, {}),
		).toEqual({ severity: 'warning' });
	});

	it('overwrites an existing managed label value (query wins)', () => {
		expect(
			mergeManagedLabels({ 'service.name': 'old' }, { 'service.name': 'new' }),
		).toEqual({ 'service.name': 'new' });
	});
});

describe('syncedLabels - syncLabelsToExpression (labels -> query)', () => {
	it('adds a new managed filter to an empty expression', () => {
		expect(syncLabelsToExpression('', { 'service.name': 'frontend' })).toBe(
			`service.name = 'frontend'`,
		);
	});

	it('appends a managed filter to an existing expression', () => {
		expect(
			syncLabelsToExpression(`http.status_code = '500'`, {
				'service.name': 'frontend',
			}),
		).toBe(`http.status_code = '500' AND service.name = 'frontend'`);
	});

	it('updates the value of an existing managed filter', () => {
		const result = syncLabelsToExpression(`service.name = 'old'`, {
			'service.name': 'new',
		});
		expect(extractManagedLabels(result)).toEqual({ 'service.name': 'new' });
	});

	it('removes a managed filter when the label is deleted', () => {
		expect(
			syncLabelsToExpression(
				`service.name = 'frontend' AND http.status_code = '500'`,
				{},
			),
		).toBe(`http.status_code = '500'`);
	});

	it('does not rewrite when nothing changed (round-trip stable)', () => {
		const expression = `service.name = 'frontend'`;
		expect(
			syncLabelsToExpression(expression, { 'service.name': 'frontend' }),
		).toBe(expression);
	});

	it('leaves non-managed clauses untouched when a managed label changes', () => {
		const result = syncLabelsToExpression(
			`http.status_code = '500' AND service.name = 'old'`,
			{ 'service.name': 'new' },
		);
		expect(result).toContain(`http.status_code = '500'`);
		expect(extractManagedLabels(result)).toEqual({ 'service.name': 'new' });
	});

	it('ignores empty-string label values (treated as absent)', () => {
		expect(syncLabelsToExpression('', { 'service.name': '' })).toBe('');
	});
});

describe('syncedLabels - isManagedLabelKey', () => {
	it('recognises managed label keys', () => {
		expect(isManagedLabelKey('service.name')).toBe(true);
		expect(isManagedLabelKey('deployment_environment')).toBe(true);
	});

	it('rejects non-managed label keys', () => {
		expect(isManagedLabelKey('severity')).toBe(false);
		expect(isManagedLabelKey('team')).toBe(false);
	});
});
