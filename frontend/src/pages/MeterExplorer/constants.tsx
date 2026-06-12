import { useTranslation } from 'react-i18next';
import { TabRoutes } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import BreakDownPage from 'container/MeterExplorer/Breakdown/BreakDown';
import ExplorerPage from 'container/MeterExplorer/Explorer';
import { Compass, TowerControl } from 'lucide-react';
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
	const { t } = useTranslation('meter');
	return (
		<div className="tab-item">
			{icon} {t(i18nKey)}
		</div>
	);
}

export const Explorer: TabRoutes = {
	Component: (): JSX.Element => <ExplorerPage />,
	name: <TabName icon={<Compass size={16} />} i18nKey="tab_explorer" />,
	route: ROUTES.METER_EXPLORER,
	key: ROUTES.METER_EXPLORER,
};

export const Views: TabRoutes = {
	Component: SaveView,
	name: <TabName icon={<TowerControl size={16} />} i18nKey="tab_views" />,
	route: ROUTES.METER_EXPLORER_VIEWS,
	key: ROUTES.METER_EXPLORER_VIEWS,
};

export const Meter: TabRoutes = {
	Component: BreakDownPage,
	name: <TabName icon={<TowerControl size={16} />} i18nKey="tab_meter" />,
	route: ROUTES.METER,
	key: ROUTES.METER,
};
