import { DEFAULT_ENTITY_VERSION } from 'constants/app';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { getVariableReferencesInQuery } from 'lib/dashboardVariables/variableReference';
import { Dashboard, WidgetRow, Widgets } from 'types/api/dashboard/getAll';

import { NocPinnedRef, NocPinnedSlot } from '../types';

export const PIN_CAP = 4;

// 핀 가능 패널 유형 — 높이 190px 스트립에 담기는 형태만 (v4 §확정 UX)
const PINNABLE_PANELS: PANEL_TYPES[] = [
	PANEL_TYPES.TIME_SERIES,
	PANEL_TYPES.BAR,
	PANEL_TYPES.VALUE,
];

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
