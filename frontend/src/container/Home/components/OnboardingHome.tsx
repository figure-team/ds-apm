import { useTranslation } from 'react-i18next';
import { Plus, Wrench } from '@signozhq/icons';
import ROUTES from 'constants/routes';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import crackerUrl from '@/assets/Icons/cracker.svg';
import dashboardUrl from '@/assets/Icons/dashboard.svg';
import wrenchUrl from '@/assets/Icons/wrench.svg';
import dottedDividerUrl from '@/assets/Images/dotted-divider.svg';

import DataSourceInfo from '../DataSourceInfo/DataSourceInfo';
import ExplorerActionCard, { ExplorerAction } from './ExplorerActionCard';

interface OnboardingHomeProps {
	isAnyIngestionActive: boolean;
	isLogsLoading: boolean;
	isTracesLoading: boolean;
}

/**
 * The onboarding home for new workspaces with no telemetry yet: data-source
 * status and explorer/dashboard/alert shortcuts. Once any signal starts
 * flowing the home switches to NocDashboard, so this view only ever renders
 * in the pre-ingestion state.
 */
export default function OnboardingHome({
	isAnyIngestionActive,
	isLogsLoading,
	isTracesLoading,
}: OnboardingHomeProps): JSX.Element {
	const { t } = useTranslation('home');
	const { user } = useAppContext();

	const explorerActions: ExplorerAction[] = [
		{
			label: t('open_logs_explorer'),
			icon: <Wrench size={14} />,
			source: 'Logs',
			route: ROUTES.LOGS_EXPLORER,
		},
		{
			label: t('open_traces_explorer'),
			icon: <Wrench size={14} />,
			source: 'Traces',
			route: ROUTES.TRACES_EXPLORER,
		},
		{
			label: t('open_metrics_explorer'),
			icon: <Wrench size={14} />,
			source: 'Metrics',
			// Distinct from the ingestion card route above — opens the explorer view.
			route: ROUTES.METRICS_EXPLORER_EXPLORER,
		},
	];

	return (
		<div className="home-left-content">
			<DataSourceInfo
				dataSentToSigNoz={isAnyIngestionActive}
				isLoading={isLogsLoading || isTracesLoading}
			/>

			<div className="divider">
				<img src={dottedDividerUrl} alt="divider" />
			</div>

			{user?.role !== USER_ROLES.VIEWER && (
				<div className="explorers-container">
					<ExplorerActionCard
						iconUrl={wrenchUrl}
						iconAlt="wrench"
						title={t('explorer_title')}
						description={t('explorer_desc')}
						actions={explorerActions}
						lazyIcon
					/>

					<ExplorerActionCard
						iconUrl={dashboardUrl}
						iconAlt="dashboard"
						title={t('dashboard_title')}
						description={t('dashboard_desc')}
						actions={[
							{
								label: t('create_dashboard'),
								icon: <Plus size={14} />,
								source: 'Dashboards',
								route: ROUTES.ALL_DASHBOARD,
							},
						]}
					/>

					<ExplorerActionCard
						iconUrl={crackerUrl}
						iconAlt="cracker"
						title={t('alert_title')}
						description={t('alert_desc')}
						actions={[
							{
								label: t('create_alert'),
								icon: <Plus size={14} />,
								source: 'Alerts',
								route: ROUTES.ALERTS_NEW,
							},
						]}
						lazyIcon
					/>
				</div>
			)}
		</div>
	);
}
