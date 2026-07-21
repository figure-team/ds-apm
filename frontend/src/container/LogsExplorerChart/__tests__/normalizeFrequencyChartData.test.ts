import { Color } from '@signozhq/design-tokens';
import { QueryData } from 'types/api/widgets/getQuery';
import { i18nText } from 'utils/i18nText';

import { getColorsForSeverityLabels, normalizeFrequencyChartData } from '../utils';

const NONE_LABEL = i18nText('common:severity_none');

function makeSeries(
	severityText: string | undefined,
	values: [number, string][],
): QueryData {
	return {
		metric:
			severityText === undefined ? {} : { severity_text: severityText },
		queryName: 'A',
		legend: '{{severity_text}}',
		values,
		queries: null,
	} as unknown as QueryData;
}

describe('normalizeFrequencyChartData', () => {
	it('merges case variants of the same severity into one series', () => {
		const result = normalizeFrequencyChartData([
			makeSeries('ERROR', [
				[100, '5'],
				[200, '3'],
			]),
			makeSeries('error', [[100, '2']]),
			makeSeries('Error', [[200, '1']]),
		]);

		expect(result).toHaveLength(1);
		expect(result[0].metric).toEqual({ severity_text: 'ERROR' });
		expect(result[0].values).toEqual([
			[100, '7'],
			[200, '4'],
		]);
	});

	it('merges known aliases into their canonical severity', () => {
		const result = normalizeFrequencyChartData([
			makeSeries('Information', [[100, '10']]),
			makeSeries('INFO', [[100, '5']]),
			makeSeries('info', [[200, '1']]),
			makeSeries('Warning', [[100, '2']]),
			makeSeries('warn', [[100, '3']]),
		]);

		const bySeverity = Object.fromEntries(
			result.map((series) => [series.metric.severity_text, series.values]),
		);

		expect(Object.keys(bySeverity).sort()).toEqual(['INFO', 'WARN']);
		expect(bySeverity.INFO).toEqual([
			[100, '15'],
			[200, '1'],
		]);
		expect(bySeverity.WARN).toEqual([[100, '5']]);
	});

	it('labels empty severity as the named "no severity" bucket', () => {
		const result = normalizeFrequencyChartData([
			makeSeries('', [[100, '41573']]),
			makeSeries('INFO', [[100, '1']]),
		]);

		const labels = result.map((series) => series.metric.severity_text);
		expect(labels).toContain(NONE_LABEL);
		expect(labels).toContain('INFO');
	});

	it('keeps unknown severities separate but case-merged', () => {
		const result = normalizeFrequencyChartData([
			makeSeries('Notice', [[100, '1']]),
			makeSeries('NOTICE', [[100, '2']]),
			makeSeries('verbose', [[100, '4']]),
		]);

		const bySeverity = Object.fromEntries(
			result.map((series) => [series.metric.severity_text, series.values]),
		);

		expect(Object.keys(bySeverity).sort()).toEqual(['NOTICE', 'VERBOSE']);
		expect(bySeverity.NOTICE).toEqual([[100, '3']]);
	});

	it('passes through series without a severity_text label (live logs)', () => {
		const plain = makeSeries(undefined, [[100, '9']]);
		const result = normalizeFrequencyChartData([plain]);

		expect(result).toHaveLength(1);
		expect(result[0]).toBe(plain);
	});
});

describe('getColorsForSeverityLabels — no-severity bucket', () => {
	it('returns the gray color for the named no-severity label', () => {
		expect(getColorsForSeverityLabels(NONE_LABEL, 0)).toBe(
			Color.BG_VANILLA_400,
		);
	});
});
