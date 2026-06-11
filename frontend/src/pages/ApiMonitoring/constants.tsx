import { TabRoutes } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import ExplorerPage from 'container/ApiMonitoring/Explorer/Explorer';
import { Compass } from 'lucide-react';
import { useTranslation } from 'react-i18next';

function ExplorerTabName(): JSX.Element {
	const { t } = useTranslation('apiMonitoring');
	return (
		<div className="tab-item">
			<Compass size={16} /> {t('tab_explorer')}
		</div>
	);
}

export const Explorer: TabRoutes = {
	Component: ExplorerPage,
	name: <ExplorerTabName />,
	route: ROUTES.API_MONITORING,
	key: ROUTES.API_MONITORING,
};
