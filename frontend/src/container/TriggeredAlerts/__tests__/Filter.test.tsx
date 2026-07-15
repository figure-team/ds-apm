import { fireEvent, render, screen } from '@testing-library/react';

import Filter from '../Filter';
import { createAlert } from './mockUtils';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

describe('Filter', () => {
	it('offers key:value autocomplete options excluding internal labels', () => {
		const allAlerts = [
			createAlert({
				labels: { severity: 'warning', ruleId: 'rule-1' },
			}),
		];

		render(
			<Filter
				allAlerts={allAlerts}
				selectedGroup={[]}
				selectedFilter={[]}
				onSelectedFilterChange={jest.fn()}
				onSelectedGroupChange={jest.fn()}
			/>,
		);

		const [filterCombobox] = screen.getAllByRole('combobox');
		fireEvent.mouseDown(filterCombobox);

		// antd Select(virtual)는 접근성을 위해 활성 옵션을 숨겨진 listbox에도
		// 동일 텍스트로 한 번 더 렌더링하므로 getByText 대신 getAllByText로 확인한다.
		expect(screen.getAllByText('severity:warning').length).toBeGreaterThan(0);
		expect(screen.queryByText('ruleId:rule-1')).not.toBeInTheDocument();
	});
});
