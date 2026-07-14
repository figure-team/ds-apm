import circusTentUrl from '@/assets/Icons/circus-tent.svg';
import eightBallUrl from '@/assets/Icons/eight-ball.svg';

const ITEM_ICONS = [circusTentUrl, eightBallUrl];

export function getItemIcon(id: string): string {
	if (!id) {
		return ITEM_ICONS[0];
	}
	return ITEM_ICONS[id.charCodeAt(id.length - 1) % ITEM_ICONS.length];
}

export const DOCS_LINKS = {
	ADD_DATA_SOURCE: 'https://signoz.io/docs/instrumentation/overview/',
	SEND_LOGS: 'https://signoz.io/docs/userguide/logs/',
	SEND_TRACES: 'https://signoz.io/docs/userguide/traces/',
	SEND_METRICS: 'https://signoz.io/docs/metrics-management/metrics-explorer/',
	SETUP_ALERTS: 'https://signoz.io/docs/userguide/alerts-management/',
	SETUP_SAVED_VIEWS:
		'https://signoz.io/docs/product-features/saved-view/#step-2-save-your-view',
	SETUP_DASHBOARDS: 'https://signoz.io/docs/userguide/manage-dashboards/',
};
