import { mapHost } from '../useNocInfra';

describe('mapHost', () => {
	it('normalizes fraction cpu/mem to percent and derives health', () => {
		expect(mapHost({ hostName: 'h1', cpu: 0.95, memory: 0.5, active: true })).toEqual({
			name: 'h1',
			cpu: 95,
			mem: 50,
			health: 'critical', // >=90
		});
		expect(mapHost({ hostName: 'h2', cpu: 0.7, memory: 0.2, active: true }).health).toBe(
			'warning', // >=65
		);
		expect(mapHost({ hostName: 'h3', cpu: 0.1, memory: 0.1, active: true }).health).toBe(
			'healthy',
		);
	});
});
