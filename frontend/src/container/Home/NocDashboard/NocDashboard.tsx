import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { Server, Siren } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import AlertsPanel from './components/AlertsPanel';
import InfraPanel from './components/InfraPanel';
import OkStrip from './components/OkStrip';
import ServiceTrendChart from './components/ServiceTrendChart';
import SummaryBand from './components/SummaryBand';
import WatchCards from './components/WatchCards';
import useNocAlerts from './hooks/useNocAlerts';
import useNocInfra from './hooks/useNocInfra';
import useNocOverview from './hooks/useNocOverview';
import useNocTrend from './hooks/useNocTrend';
import { TrendMetric } from './types';
import {
	deriveCounts,
	pickIncident,
	selectTrendTargets,
	selectWatch,
} from './utils/deriveState';

import './NocDashboard.styles.scss';

const TREND_WATCH_THRESHOLD = 1; // 정상 모드 오류율 주의 임계 1% (§4.3)

export default function NocDashboard(): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { t } = useTranslation('home');
	const [metric, setMetric] = useState<TrendMetric>('err');

	const {
		alerts,
		firingCount,
		isLoading: alertsLoading,
		isError: alertsError,
		lastResolved,
	} = useNocAlerts();
	const overview = useNocOverview(firingCount);
	const { services } = overview;

	const counts = useMemo(() => deriveCounts(services, firingCount), [
		services,
		firingCount,
	]);
	const watch = useMemo(() => selectWatch(services), [services]);
	const targets = useMemo(() => selectTrendTargets(services), [services]);
	const incident = useMemo(() => pickIncident(alerts), [alerts]);

	const trend = useNocTrend(targets, metric);
	const infra = useNocInfra();

	const criticalOverflow = Math.max(0, counts.critical - watch.services.length);
	const okNames = useMemo(
		() => services.filter((s) => s.health === 'healthy').map((s) => s.name),
		[services],
	);
	const anomaly = counts.critical > 0 || counts.warning > 0 || counts.alerts > 0;

	return (
		<div className={`noc-root noc-c2 ${isDarkMode ? 'noc-dark' : 'noc-light'}`}>
			<div className="noc-toolbar">
				<div className="noc-live">
					<span className="noc-live-pulse" />
					{t('noc_live_ingesting')}
				</div>
				<div className="noc-toolbar-spacer" />
				<div className="noc-time-select">
					<DateTimeSelectionV2
						showAutoRefresh
						showRefreshText={false}
						hideShareModal
						defaultRelativeTime="30m"
					/>
				</div>
			</div>

			<SummaryBand counts={counts} incident={incident} />

			<div className="noc-c2-body">
				<div className="noc-c2-left">
					<WatchCards
						services={watch.services}
						mode={watch.mode}
						overflowCount={criticalOverflow}
					/>
					<div className="noc-trend-wrap">
						<ServiceTrendChart
							series={trend.series}
							metric={metric}
							onMetricChange={setMetric}
							thresholdLine={!anomaly ? TREND_WATCH_THRESHOLD : undefined}
							loading={trend.isLoading}
							error={trend.isError}
						/>
					</div>
					<OkStrip names={okNames} />
				</div>
				<div className="noc-c2-right">
					<section className="noc-c2-panel">
						<div className="noc-c2-panel-head">
							<Siren size={13} />
							<span>{t('noc_c2_alerts_title')}</span>
						</div>
						<div className="noc-c2-panel-body">
							<AlertsPanel
								alerts={alerts}
								isLoading={alertsLoading}
								isError={alertsError}
								lastResolved={lastResolved}
							/>
						</div>
					</section>
					<section className="noc-c2-panel">
						<div className="noc-c2-panel-head">
							<Server size={13} />
							<span>{t('noc_c2_infra_title')}</span>
						</div>
						<div className="noc-c2-panel-body">
							<InfraPanel
								hosts={infra.hosts}
								isLoading={infra.isLoading}
								isError={infra.isError}
							/>
						</div>
					</section>
				</div>
			</div>
		</div>
	);
}
