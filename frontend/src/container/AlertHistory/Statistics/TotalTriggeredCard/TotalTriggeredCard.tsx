import { useTranslation } from 'react-i18next';
import { AlertRuleStats } from 'types/api/alerts/def';

import StatsCard from '../StatsCard/StatsCard';

type TotalTriggeredCardProps = {
	totalCurrentTriggers: AlertRuleStats['totalCurrentTriggers'];
	totalPastTriggers: AlertRuleStats['totalPastTriggers'];
	timeSeries: AlertRuleStats['currentTriggersSeries']['values'];
};

function TotalTriggeredCard({
	totalCurrentTriggers,
	totalPastTriggers,
	timeSeries,
}: TotalTriggeredCardProps): JSX.Element {
	const { t } = useTranslation('alerts');
	return (
		<StatsCard
			totalCurrentCount={totalCurrentTriggers}
			totalPastCount={totalPastTriggers}
			title={t('hist_total_triggered')}
			timeSeries={timeSeries}
		/>
	);
}

export default TotalTriggeredCard;
