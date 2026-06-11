import { useTranslation } from 'react-i18next';
import { TabRoutes } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import ExplorerPage from 'container/MetricsExplorer/Explorer';
import SummaryPage from 'container/MetricsExplorer/Summary';
import { BarChart2, Compass, TowerControl } from 'lucide-react';
import SaveView from 'pages/SaveView';

// Tab labels live in components so useTranslation can run at render time
// (TabRoutes objects are created at module scope where hooks are unavailable).
function TabName({
	icon,
	i18nKey,
}: {
	icon: JSX.Element;
	i18nKey: string;
}): JSX.Element {
	const { t } = useTranslation('metricsExplorer');
	return (
		<div className="tab-item">
			{icon} {t(i18nKey)}
		</div>
	);
}

export const Summary: TabRoutes = {
	Component: SummaryPage,
	name: <TabName icon={<BarChart2 size={16} />} i18nKey="tab_summary" />,
	route: ROUTES.METRICS_EXPLORER,
	key: ROUTES.METRICS_EXPLORER,
};

export const Explorer: TabRoutes = {
	Component: (): JSX.Element => <ExplorerPage />,
	name: <TabName icon={<Compass size={16} />} i18nKey="tab_explorer" />,
	route: ROUTES.METRICS_EXPLORER_EXPLORER,
	key: ROUTES.METRICS_EXPLORER_EXPLORER,
};

export const Views: TabRoutes = {
	Component: SaveView,
	name: <TabName icon={<TowerControl size={16} />} i18nKey="tab_views" />,
	route: ROUTES.METRICS_EXPLORER_VIEWS,
	key: ROUTES.METRICS_EXPLORER_VIEWS,
};
