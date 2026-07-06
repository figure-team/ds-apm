import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import FiredAlertsBadge from './components/FiredAlertsBadge';
import InfraBadge from './components/InfraBadge';
import PinnedPanels from './components/PinnedPanels';
import PinPickerDrawer from './components/PinPickerDrawer';
import ServiceTrendChart from './components/ServiceTrendChart';
import SummaryBand from './components/SummaryBand';
import WatchCards from './components/WatchCards';
import useNocAlerts from './hooks/useNocAlerts';
import useNocInfra from './hooks/useNocInfra';
import useNocOverview from './hooks/useNocOverview';
import useNocPinnedPanels from './hooks/useNocPinnedPanels';
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
	const [pickerOpen, setPickerOpen] = useState(false);

	const { alerts, firingCount } = useNocAlerts();
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
	const pinned = useNocPinnedPanels();

	const anomaly = counts.critical > 0 || counts.warning > 0 || counts.alerts > 0;
	const criticalOverflow = Math.max(0, counts.critical - watch.services.length);

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

			<SummaryBand
				counts={counts}
				incident={incident}
				actions={
					<>
						<FiredAlertsBadge count={firingCount} />
						<InfraBadge
							hosts={infra.hosts}
							isLoading={infra.isLoading}
							isError={infra.isError}
						/>
					</>
				}
			/>

			<div className="noc-c2-main">
				{watch.mode === 'anomaly' ? (
					<div className="noc-c2-watch-transient">
						<WatchCards
							services={watch.services}
							mode="anomaly"
							overflowCount={criticalOverflow}
						/>
					</div>
				) : null}
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
				<PinnedPanels
					slots={pinned.slots}
					onUnpin={pinned.unpin}
					onOpenPicker={(): void => setPickerOpen(true)}
				/>
			</div>

			<PinPickerDrawer
				open={pickerOpen}
				onClose={(): void => setPickerOpen(false)}
				dashboards={pinned.dashboards}
				refs={pinned.refs}
				onPin={pinned.pin}
				onUnpin={pinned.unpin}
			/>
		</div>
	);
}
