import { fireEvent, render, screen } from '@testing-library/react';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { Dashboard } from 'types/api/dashboard/getAll';

import PinPickerDrawer from '../PinPickerDrawer';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

const DASH = ({
	id: 'd1',
	createdAt: '',
	updatedAt: '',
	createdBy: '',
	updatedBy: '',
	data: {
		title: '결제 구간 모니터링',
		variables: {},
		version: 'v5',
		widgets: [
			{
				id: 'w1',
				panelTypes: PANEL_TYPES.TIME_SERIES,
				title: '결제 지연 p95',
				description: '',
				opacity: '1',
				nullZeroValues: '',
				timePreferance: 'GLOBAL_TIME',
				softMin: null,
				softMax: null,
				selectedLogFields: null,
				selectedTracesFields: null,
				query: {
					queryType: 'builder',
					promql: [],
					clickhouse_sql: [],
					builder: { queryData: [], queryFormulas: [], queryTraceOperator: [] },
					id: 'q1',
				},
			},
		],
	},
} as unknown) as Dashboard;

describe('PinPickerDrawer', () => {
	it('lists pinnable widgets and pins on check', () => {
		const onPin = jest.fn();
		render(
			<PinPickerDrawer
				open
				onClose={jest.fn()}
				dashboards={[DASH]}
				refs={[]}
				onPin={onPin}
				onUnpin={jest.fn()}
			/>,
		);
		// 대시보드 아코디언 펼치기
		fireEvent.click(screen.getByText('결제 구간 모니터링'));
		fireEvent.click(screen.getByLabelText('결제 지연 p95'));
		expect(onPin).toHaveBeenCalledWith({ dashboardId: 'd1', widgetId: 'w1' });
	});

	it('unpins on uncheck', () => {
		const onUnpin = jest.fn();
		render(
			<PinPickerDrawer
				open
				onClose={jest.fn()}
				dashboards={[DASH]}
				refs={[{ dashboardId: 'd1', widgetId: 'w1' }]}
				onPin={jest.fn()}
				onUnpin={onUnpin}
			/>,
		);
		fireEvent.click(screen.getByText('결제 구간 모니터링'));
		fireEvent.click(screen.getByLabelText('결제 지연 p95'));
		expect(onUnpin).toHaveBeenCalledWith('w1');
	});
});
