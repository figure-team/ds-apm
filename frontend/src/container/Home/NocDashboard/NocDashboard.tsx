import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useTranslation } from 'react-i18next';

import InsightsColumn from './components/InsightsColumn';
import LeftRail from './components/LeftRail';
import ServiceHealthList from './components/ServiceHealthList';
import useNocAlerts from './hooks/useNocAlerts';
import useNocLogs from './hooks/useNocLogs';
import useNocOverview from './hooks/useNocOverview';
import useNocRca from './hooks/useNocRca';

import './NocDashboard.styles.scss';

export default function NocDashboard(): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { t } = useTranslation('home');
	const { alerts, firingCount, isLoading, isError } = useNocAlerts();
	const overview = useNocOverview(firingCount);
	const rca = useNocRca();
	const logs = useNocLogs();

	return (
		<div className={`noc-root ${isDarkMode ? 'noc-dark' : 'noc-light'}`}>
			<div className="noc-toolbar">
				<div className="noc-env-tabs">
					<button type="button" className="active">
						PROD
					</button>
					<button type="button">DEV</button>
				</div>
				<div className="noc-toolbar-spacer" />
				<div className="noc-live">
					<span className="noc-live-pulse" />
					{t('noc_live_ingesting')}
				</div>
				<div className="noc-time-select">
					<DateTimeSelectionV2
						showAutoRefresh
						showRefreshText={false}
						hideShareModal
						defaultRelativeTime="30m"
					/>
				</div>
			</div>

			<div className="noc-console">
				<LeftRail
					kpis={overview.kpis}
					alerts={alerts}
					alertsLoading={isLoading}
					alertsError={isError}
				/>
				<ServiceHealthList
					rows={overview.services}
					isLoading={overview.isLoading}
					isError={overview.isError}
				/>
				<InsightsColumn
					rca={rca.rca}
					rcaLoading={rca.isLoading}
					rcaError={rca.isError}
					logs={logs.logs}
					logsLoading={logs.isLoading}
					logsError={logs.isError}
				/>
			</div>
		</div>
	);
}
