import { DEFAULT_ENTITY_VERSION } from 'constants/app';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { getVariableReferencesInQuery } from 'lib/dashboardVariables/variableReference';
import { Dashboard, WidgetRow, Widgets } from 'types/api/dashboard/getAll';

import { NocPinnedRef, NocPinnedSlot } from '../types';

export const PIN_CAP = 4;

// 핀 가능 패널 유형 — 좌우 2단(C-2) 재구조로 전 유형 허용.
// 오른쪽 열이 자연 높이로 세로 스크롤하므로 표·리스트·파이·히스토그램·트레이스도 안 찌그러진다.
// EMPTY_WIDGET만 제외(렌더 대상 아님). $var·row 게이트는 isPinnable에서 별도 유지.
const PINNABLE_PANELS: PANEL_TYPES[] = [
	PANEL_TYPES.TIME_SERIES,
	PANEL_TYPES.BAR,
	PANEL_TYPES.VALUE,
	PANEL_TYPES.TABLE,
	PANEL_TYPES.LIST,
	PANEL_TYPES.TRACE,
	PANEL_TYPES.PIE,
	PANEL_TYPES.HISTOGRAM,
];

// 유형별 자연 표시 높이(px) — 오른쪽 열 세로 스택에서 카드 높이로 사용(설계 §2).
// 큰 숫자(VALUE)는 낮게, 표/리스트/트레이스는 내부 스크롤 여지를 위해 높게.
export function panelDisplayHeight(panelType: PANEL_TYPES): number {
	switch (panelType) {
		case PANEL_TYPES.VALUE:
			return 120;
		case PANEL_TYPES.TABLE:
		case PANEL_TYPES.LIST:
		case PANEL_TYPES.TRACE:
			return 280;
		case PANEL_TYPES.TIME_SERIES:
		case PANEL_TYPES.BAR:
		case PANEL_TYPES.PIE:
		case PANEL_TYPES.HISTOGRAM:
			return 220;
		default:
			return 220;
	}
}

function isWidget(w: WidgetRow | Widgets): w is Widgets {
	return 'query' in w && w.query != null;
}

// 대시보드 변수($var)를 참조하는 위젯은 홈에서 변수 해석이 불가하므로 제외
export function isPinnable(
	dashboard: Dashboard,
	widget: WidgetRow | Widgets,
): boolean {
	if (!isWidget(widget)) return false;
	if (!PINNABLE_PANELS.includes(widget.panelTypes)) return false;
	const varNames = Object.values(dashboard.data.variables ?? {})
		.map((v) => v.name)
		.filter((n): n is string => Boolean(n));
	if (varNames.length === 0) return true;
	return getVariableReferencesInQuery(widget.query, varNames).length === 0;
}

export function listPinnableWidgets(dashboard: Dashboard): Widgets[] {
	return (dashboard.data.widgets ?? []).filter((w): w is Widgets =>
		isPinnable(dashboard, w),
	);
}

export function resolvePinnedSlots(
	dashboards: Dashboard[],
	refs: NocPinnedRef[],
): NocPinnedSlot[] {
	return refs.slice(0, PIN_CAP).map((ref) => {
		const dashboard = dashboards.find((d) => d.id === ref.dashboardId);
		const raw = dashboard
			? (dashboard.data.widgets ?? []).find((w) => w.id === ref.widgetId)
			: undefined;
		const ok = dashboard && raw && isPinnable(dashboard, raw);
		return {
			ref,
			dashboardTitle: dashboard?.data.title ?? '',
			version: dashboard?.data.version ?? DEFAULT_ENTITY_VERSION,
			widget: ok ? (raw as Widgets) : null,
		};
	});
}
