import { TabRoutes } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import { TFunction } from 'i18next';
import { Compass, TowerControl, Workflow } from 'lucide-react';
import LogsExplorer from 'pages/LogsExplorer';
import Pipelines from 'pages/Pipelines';
import SaveView from 'pages/SaveView';

export const logsExplorer = (t: TFunction): TabRoutes => ({
	Component: (): JSX.Element => <LogsExplorer />,
	name: (
		<div className="tab-item">
			<Compass size={16} /> {t('tab_explorer').toString()}
		</div>
	),
	route: ROUTES.LOGS,
	key: ROUTES.LOGS,
});

export const logsPipelines = (t: TFunction): TabRoutes => ({
	Component: (): JSX.Element => <Pipelines />,
	name: (
		<div className="tab-item">
			<Workflow size={16} /> {t('tab_pipelines').toString()}
		</div>
	),
	route: ROUTES.LOGS_PIPELINES,
	key: ROUTES.LOGS_PIPELINES,
});

export const logSaveView = (t: TFunction): TabRoutes => ({
	Component: SaveView,
	name: (
		<div className="tab-item">
			<TowerControl size={16} /> {t('tab_views').toString()}
		</div>
	),
	route: ROUTES.LOGS_SAVE_VIEWS,
	key: ROUTES.LOGS_SAVE_VIEWS,
});
