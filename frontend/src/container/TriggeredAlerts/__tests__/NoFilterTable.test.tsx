import { fireEvent, render, screen } from '@testing-library/react';

import NoFilterTable from '../NoFilterTable';
import { createAlert } from './mockUtils';

jest.mock('providers/Timezone', () => ({
	useTimezone: jest.requireActual('./mockUtils').useMockTimezone,
}));

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

const allAlerts = [
	createAlert({
		name: 'Alert B',
		labels: {
			severity: 'warning',
			alertname: 'Alert B',
		},
	}),
	createAlert({
		name: 'Alert C',
		labels: {
			severity: 'info',
			alertname: 'Alert C',
		},
	}),
	createAlert({
		name: 'Alert A',
		labels: {
			severity: 'critical',
			alertname: 'Alert A',
		},
	}),
];

describe('NoFilterTable', () => {
	it('should render the no filter table with correct rows', () => {
		render(<NoFilterTable allAlerts={allAlerts} selectedFilter={[]} />);
		const rows = screen.getAllByRole('row');
		expect(rows).toHaveLength(4); // 1 header row + 2 data rows
		const [headerRow, dataRow1, dataRow2, dataRow3] = rows;

		// Verify header row
		expect(headerRow).toHaveTextContent('column_status');
		expect(headerRow).toHaveTextContent('column_alert_name');
		expect(headerRow).toHaveTextContent('triggered_column_tags');
		expect(headerRow).toHaveTextContent('column_severity');
		expect(headerRow).toHaveTextContent('triggered_column_firing_since');

		// Verify 1st data row
		expect(dataRow1).toHaveTextContent('Alert B');

		// Verify 2nd data row
		expect(dataRow2).toHaveTextContent('Alert C');

		// Verify 3rd data row
		expect(dataRow3).toHaveTextContent('Alert A');
	});

	it('should sort the table by severity when header is clicked', () => {
		render(<NoFilterTable allAlerts={allAlerts} selectedFilter={[]} />);

		const headers = screen.getAllByRole('columnheader');
		const severityHeader = headers.find((header) =>
			header.textContent?.includes('column_severity'),
		);

		expect(severityHeader).toBeInTheDocument();

		if (severityHeader) {
			const initialRows = screen.getAllByRole('row');
			expect(initialRows.length).toBe(4);
			expect(initialRows[1]).toHaveTextContent('Alert B');
			expect(initialRows[2]).toHaveTextContent('Alert C');
			expect(initialRows[3]).toHaveTextContent('Alert A');

			fireEvent.click(severityHeader);

			const sortedRows = screen.getAllByRole('row');
			expect(sortedRows.length).toBe(4);
			expect(sortedRows[1]).toHaveTextContent('Alert A');
			expect(sortedRows[2]).toHaveTextContent('Alert B');
			expect(sortedRows[3]).toHaveTextContent('Alert C');
		}
	});
});
