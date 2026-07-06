import { PANEL_TYPES } from 'constants/queryBuilder';
import { Dashboard, Widgets } from 'types/api/dashboard/getAll';

import {
	isPinnable,
	listPinnableWidgets,
	PIN_CAP,
	resolvePinnedSlots,
} from '../pinnedPanels';

function makeWidget(over: Partial<Widgets>): Widgets {
	return ({
		id: 'w1',
		panelTypes: PANEL_TYPES.TIME_SERIES,
		title: 'w',
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
		...over,
	} as unknown) as Widgets;
}

function makeDashboard(
	id: string,
	widgets: Widgets[],
	variables: Record<string, { name?: string }> = {},
): Dashboard {
	return ({
		id,
		createdAt: '',
		updatedAt: '',
		createdBy: '',
		updatedBy: '',
		data: {
			title: `dash-${id}`,
			widgets,
			variables,
			version: 'v5',
		},
	} as unknown) as Dashboard;
}

describe('pinnedPanels utils', () => {
	it('accepts graph/bar/value panels, rejects table/list/row', () => {
		const d = makeDashboard('d1', []);
		expect(isPinnable(d, makeWidget({ panelTypes: PANEL_TYPES.TIME_SERIES }))).toBe(
			true,
		);
		expect(isPinnable(d, makeWidget({ panelTypes: PANEL_TYPES.BAR }))).toBe(true);
		expect(isPinnable(d, makeWidget({ panelTypes: PANEL_TYPES.VALUE }))).toBe(
			true,
		);
		expect(isPinnable(d, makeWidget({ panelTypes: PANEL_TYPES.TABLE }))).toBe(
			false,
		);
		expect(isPinnable(d, makeWidget({ panelTypes: PANEL_TYPES.LIST }))).toBe(
			false,
		);
	});

	it('rejects row entries that have no query (WidgetRow)', () => {
		const d = makeDashboard('d1', []);
		// WidgetRow — query 없음, panelTypes 'row'
		const row = ({ id: 'r1', panelTypes: 'row' } as unknown) as Widgets;
		expect(isPinnable(d, row)).toBe(false);
	});

	it('rejects widgets whose query references a dashboard variable', () => {
		const w = makeWidget({
			query: ({
				queryType: 'builder',
				promql: [],
				clickhouse_sql: [],
				builder: {
					queryData: [{ filter: { expression: 'service.name = $svc' } }],
					queryFormulas: [],
					queryTraceOperator: [],
				},
				id: 'q2',
			} as unknown) as Widgets['query'],
		});
		const d = makeDashboard('d1', [w], { v1: { name: 'svc' } });
		expect(isPinnable(d, w)).toBe(false);
		expect(listPinnableWidgets(d)).toHaveLength(0);
	});

	it('keeps widgets that do not reference any dashboard variable', () => {
		const w = makeWidget({
			query: ({
				queryType: 'builder',
				promql: [],
				clickhouse_sql: [],
				builder: {
					queryData: [{ filter: { expression: "service.name = 'checkout'" } }],
					queryFormulas: [],
					queryTraceOperator: [],
				},
				id: 'q3',
			} as unknown) as Widgets['query'],
		});
		const d = makeDashboard('d1', [w], { v1: { name: 'svc' } });
		expect(isPinnable(d, w)).toBe(true);
		expect(listPinnableWidgets(d)).toHaveLength(1);
	});

	it('rejects promql widgets referencing a variable', () => {
		const w = makeWidget({
			query: ({
				queryType: 'promql',
				promql: [{ query: 'rate(http_requests{svc="$svc"}[5m])', name: 'A' }],
				clickhouse_sql: [],
				builder: { queryData: [], queryFormulas: [], queryTraceOperator: [] },
				id: 'q4',
			} as unknown) as Widgets['query'],
		});
		const d = makeDashboard('d1', [w], { v1: { name: 'svc' } });
		expect(isPinnable(d, w)).toBe(false);
	});

	it('resolves refs to slots, null widget for deleted dashboards, caps at PIN_CAP', () => {
		const w = makeWidget({ id: 'w1' });
		const d = makeDashboard('d1', [w]);
		const refs = [
			{ dashboardId: 'd1', widgetId: 'w1' },
			{ dashboardId: 'gone', widgetId: 'wx' },
			{ dashboardId: 'd1', widgetId: 'w1' },
			{ dashboardId: 'd1', widgetId: 'w1' },
			{ dashboardId: 'd1', widgetId: 'w1' },
		];
		const slots = resolvePinnedSlots([d], refs);
		expect(slots).toHaveLength(PIN_CAP);
		expect(slots[0].widget?.id).toBe('w1');
		expect(slots[0].dashboardTitle).toBe('dash-d1');
		expect(slots[0].version).toBe('v5');
		expect(slots[1].widget).toBeNull();
		expect(slots[1].dashboardTitle).toBe('');
	});

	it('nulls the widget when it exists but is no longer pinnable', () => {
		const w = makeWidget({ id: 'w1', panelTypes: PANEL_TYPES.TABLE });
		const d = makeDashboard('d1', [w]);
		const slots = resolvePinnedSlots([d], [{ dashboardId: 'd1', widgetId: 'w1' }]);
		expect(slots).toHaveLength(1);
		expect(slots[0].widget).toBeNull();
		expect(slots[0].dashboardTitle).toBe('dash-d1');
	});
});
