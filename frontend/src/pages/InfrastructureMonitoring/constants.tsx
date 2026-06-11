import { useTranslation } from 'react-i18next';
import { TabRoutes } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import InfraMonitoringHosts from 'container/InfraMonitoringHosts';
import InfraMonitoringK8s from 'container/InfraMonitoringK8s';
import { Inbox } from 'lucide-react';

// Tab labels live in components so useTranslation can run at render time
// (TabRoutes objects are created at module scope where hooks are unavailable).
function TabName({ i18nKey }: { i18nKey: string }): JSX.Element {
	const { t } = useTranslation('infraMonitoring');
	return (
		<div className="tab-item">
			<Inbox size={16} /> {t(i18nKey)}
		</div>
	);
}

export const Hosts: TabRoutes = {
	Component: (): JSX.Element => <InfraMonitoringHosts />,
	name: <TabName i18nKey="tab_hosts" />,
	route: ROUTES.INFRASTRUCTURE_MONITORING_HOSTS,
	key: ROUTES.INFRASTRUCTURE_MONITORING_HOSTS,
};

export const Kubernetes: TabRoutes = {
	Component: (): JSX.Element => <InfraMonitoringK8s />,
	name: <TabName i18nKey="tab_kubernetes" />,
	route: ROUTES.INFRASTRUCTURE_MONITORING_KUBERNETES,
	key: ROUTES.INFRASTRUCTURE_MONITORING_KUBERNETES,
};
