import { useTranslation } from 'react-i18next';
import { Plus, Wrench } from '@signozhq/icons';
import ROUTES from 'constants/routes';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import crackerUrl from '@/assets/Icons/cracker.svg';
import dashboardUrl from '@/assets/Icons/dashboard.svg';
import wrenchUrl from '@/assets/Icons/wrench.svg';
import dottedDividerUrl from '@/assets/Images/dotted-divider.svg';

import AlertRules from '../AlertRules/AlertRules';
import Dashboards from '../Dashboards/Dashboards';
import DataSourceInfo from '../DataSourceInfo/DataSourceInfo';
import SavedViews from '../SavedViews/SavedViews';
import Services from '../Services/Services';
import ActiveIngestionCard from './ActiveIngestionCard';
import ExplorerActionCard, { ExplorerAction } from './ExplorerActionCard';

interface OnboardingHomeProps {
	isLogsIngestionActive: boolean;
	isTracesIngestionActive: boolean;
	isMetricsIngestionActive: boolean;
	isAnyIngestionActive: boolean;
	isLogsLoading: boolean;
	isTracesLoading: boolean;
}

/**
 * The pre-NOC onboarding home: ingestion status, explorer shortcuts and the
 * alert/dashboard/services panels. Rendered until telemetry starts flowing.
 */
export default function OnboardingHome({
	isLogsIngestionActive,
	isTracesIngestionActive,
	isMetricsIngestionActive,
	isAnyIngestionActive,
	isLogsLoading,
	isTracesLoading,
}: OnboardingHomeProps): JSX.Element {
	const { t } = useTranslation('home');
	const { user } = useAppContext();

	const activeIngestions = [
		{
			isActive: isLogsIngestionActive,
			description: t('ingestion_active_logs'),
			exploreLabel: t('explore_logs'),
			source: 'Logs',
			route: ROUTES.LOGS_EXPLORER,
		},
		{
			isActive: isTracesIngestionActive,
			description: t('ingestion_active_traces'),
			exploreLabel: t('explore_traces'),
			source: 'Traces',
			route: ROUTES.TRACES_EXPLORER,
		},
		{
			isActive: isMetricsIngestionActive,
			description: t('ingestion_active_metrics'),
			exploreLabel: t('explore_metrics'),
			source: 'Metrics',
			// Explore goes to the metrics summary, unlike the explorer button below.
			route: ROUTES.METRICS_EXPLORER,
		},
	];

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
		<>
			<div className="home-left-content">
				<DataSourceInfo
					dataSentToSigNoz={isAnyIngestionActive}
					isLoading={isLogsLoading || isTracesLoading}
				/>

				<div className="divider">
					<img src={dottedDividerUrl} alt="divider" />
				</div>

				<div className="active-ingestions-container">
					{activeIngestions.map(
						(ingestion) =>
							ingestion.isActive && (
								<ActiveIngestionCard
									key={ingestion.source}
									description={ingestion.description}
									exploreLabel={ingestion.exploreLabel}
									source={ingestion.source}
									route={ingestion.route}
								/>
							),
					)}
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

				{isAnyIngestionActive && (
					<>
						<AlertRules />
						<Dashboards />
					</>
				)}
			</div>
			<div className="home-right-content">
				{isAnyIngestionActive && (
					<>
						<Services />
						<SavedViews />
					</>
				)}
			</div>
		</>
	);
}
