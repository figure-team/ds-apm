import {
	getMissingOperationalLabels,
	hasOperationalLabel,
	RECOMMENDED_OPERATIONAL_LABELS,
} from '../operationalMetadata';

const ALL_FILLED_LABELS = {
	environment: 'prod',
	owner_team: 'sm-payments',
	project_id: 'customer-a',
	'service.name': 'payment-api',
};

describe('operationalMetadata', () => {
	it('reports every recommended SI/SM label as missing when metadata is empty', () => {
		expect(getMissingOperationalLabels({})).toStrictEqual(
			RECOMMENDED_OPERATIONAL_LABELS,
		);
	});

	it('treats blank label values as missing', () => {
		expect(
			getMissingOperationalLabels({
				environment: ' ',
				owner_team: 'sm-payments',
				project_id: 'customer-a',
				'service.name': 'payment-api',
			}).map(({ key }) => key),
		).toStrictEqual(['environment']);
	});

	it('detects complete SI/SM routing labels', () => {
		expect(getMissingOperationalLabels(ALL_FILLED_LABELS)).toStrictEqual([]);
		expect(hasOperationalLabel(ALL_FILLED_LABELS, 'service.name')).toBe(true);
	});

	describe('missing → filled status transitions', () => {
		it('transitions a single label from missing to filled', () => {
			const before = {};
			const after = { owner_team: 'sm-payments' };

			expect(hasOperationalLabel(before, 'owner_team')).toBe(false);
			expect(hasOperationalLabel(after, 'owner_team')).toBe(true);

			const missingBefore = getMissingOperationalLabels(before).map(({ key }) => key);
			const missingAfter = getMissingOperationalLabels(after).map(({ key }) => key);

			expect(missingBefore).toContain('owner_team');
			expect(missingAfter).not.toContain('owner_team');
		});

		it('reduces missing count as labels are filled one by one', () => {
			const labels: Record<string, string> = {};
			const keys = RECOMMENDED_OPERATIONAL_LABELS.map(({ key }) => key);

			for (let i = 0; i < keys.length; i++) {
				expect(getMissingOperationalLabels(labels)).toHaveLength(keys.length - i);
				labels[keys[i]] = 'value';
			}

			expect(getMissingOperationalLabels(labels)).toHaveLength(0);
		});

		it('reverts to missing when a label value is cleared', () => {
			expect(hasOperationalLabel({ environment: 'prod' }, 'environment')).toBe(true);
			expect(hasOperationalLabel({ environment: '' }, 'environment')).toBe(false);
			expect(hasOperationalLabel({ environment: '   ' }, 'environment')).toBe(false);
		});

		it('reaches zero missing labels once all 4 are filled', () => {
			const partialLabels = {
				project_id: 'customer-a',
				environment: 'prod',
				'service.name': 'payment-api',
			};
			expect(getMissingOperationalLabels(partialLabels)).toHaveLength(1);

			const completeLabels = {
				...partialLabels,
				owner_team: 'sm-payments',
			};
			expect(getMissingOperationalLabels(completeLabels)).toHaveLength(0);
		});
	});
});
