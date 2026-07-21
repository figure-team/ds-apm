import { render, screen } from '@testing-library/react';

import EmptyMetricsSearch from '../EmptyMetricsSearch';

describe('EmptyMetricsSearch', () => {
	it('shows select metric message when no query has been run', () => {
		render(<EmptyMetricsSearch />);

		expect(
			screen.getByText('metricsExplorer:empty_select_metric'),
		).toBeInTheDocument();
	});

	it('shows no data message when a query returned empty results', () => {
		render(<EmptyMetricsSearch hasQueryResult />);

		expect(screen.getByText('common:no_data')).toBeInTheDocument();
	});
});
