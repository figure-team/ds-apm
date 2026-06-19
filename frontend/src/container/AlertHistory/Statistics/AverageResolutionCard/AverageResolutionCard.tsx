import { useTranslation } from 'react-i18next';
import { AlertRuleStats } from 'types/api/alerts/def';
import { formatTime } from 'utils/timeUtils';

import StatsCard from '../StatsCard/StatsCard';

type TotalTriggeredCardProps = {
	currentAvgResolutionTime: AlertRuleStats['currentAvgResolutionTime'];
	pastAvgResolutionTime: AlertRuleStats['pastAvgResolutionTime'];
	timeSeries: AlertRuleStats['currentAvgResolutionTimeSeries']['values'];
};

function AverageResolutionCard({
	currentAvgResolutionTime,
	pastAvgResolutionTime,
	timeSeries,
}: TotalTriggeredCardProps): JSX.Element {
	const { t } = useTranslation('alerts');
	return (
		<StatsCard
			displayValue={formatTime(+currentAvgResolutionTime)}
			totalCurrentCount={+currentAvgResolutionTime}
			totalPastCount={+pastAvgResolutionTime}
			title={t('hist_avg_resolution_time')}
			timeSeries={timeSeries}
		/>
	);
}

export default AverageResolutionCard;
