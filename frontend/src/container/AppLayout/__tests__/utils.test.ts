import { getRouteKey } from '../utils';

describe('getRouteKey', () => {
	it('matches static routes exactly', () => {
		expect(getRouteKey('/services')).toBe('APPLICATION');
		expect(getRouteKey('/service-map')).toBe('SERVICE_MAP');
		expect(getRouteKey('/traces-explorer')).toBe('TRACES_EXPLORER');
	});

	it('matches dynamic routes by pattern', () => {
		expect(getRouteKey('/services/petclinic')).toBe('SERVICE_METRICS');
		expect(getRouteKey('/trace/2a0530b4693bf8df04c2430b2ee35efd')).toBe(
			'TRACE_DETAIL',
		);
		expect(getRouteKey('/services/petclinic/top-level-operations')).toBe(
			'SERVICE_TOP_LEVEL_OPERATIONS',
		);
	});

	it('falls back to SETTINGS for unlisted settings subpaths', () => {
		expect(getRouteKey('/settings/general')).toBe('SETTINGS');
		expect(getRouteKey('/settings/sop-documents')).toBe('SOP_DOCUMENTS_SETTINGS');
	});

	it('falls back to DEFAULT for unknown paths', () => {
		expect(getRouteKey('/definitely-not-a-route')).toBe('DEFAULT');
	});
});
