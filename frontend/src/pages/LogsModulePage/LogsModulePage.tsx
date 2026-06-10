import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-use';
import RouteTab from 'components/RouteTab';
import { TabRoutes } from 'components/RouteTab/types';
import history from 'lib/history';

import { logSaveView, logsExplorer, logsPipelines } from './constants';

import './LogsModulePage.styles.scss';

export default function LogsModulePage(): JSX.Element {
	const { pathname } = useLocation();
	const { t } = useTranslation(['logs']);

	const routes: TabRoutes[] = [
		logsExplorer(t),
		logsPipelines(t),
		logSaveView(t),
	];

	return (
		<div className="logs-module-container">
			<RouteTab routes={routes} activeKey={pathname} history={history} />
		</div>
	);
}
