import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useGetAlertRuleDetailsStats } from 'pages/AlertDetails/hooks';
import DataStateRenderer from 'periscope/components/DataStateRenderer/DataStateRenderer';

import AverageResolutionCard from '../AverageResolutionCard/AverageResolutionCard';
import StatsCard from '../StatsCard/StatsCard';
import TotalTriggeredCard from '../TotalTriggeredCard/TotalTriggeredCard';

const hasTotalTriggeredStats = (
	totalCurrentTriggers: number | string,
	totalPastTriggers: number | string,
): boolean =>
	(Number(totalCurrentTriggers) > 0 && Number(totalPastTriggers) > 0) ||
	Number(totalCurrentTriggers) > 0;

const hasAvgResolutionTimeStats = (
	currentAvgResolutionTime: number | string,
	pastAvgResolutionTime: number | string,
): boolean =>
	(Number(currentAvgResolutionTime) > 0 && Number(pastAvgResolutionTime) > 0) ||
	Number(currentAvgResolutionTime) > 0;

type StatsCardsRendererProps = {
	setTotalCurrentTriggers: (value: number) => void;
};

// TODO(shaheer): render the DataStateRenderer inside the TotalTriggeredCard/AverageResolutionCard, it should display the title
function StatsCardsRenderer({
	setTotalCurrentTriggers,
}: StatsCardsRendererProps): JSX.Element {
	const { t } = useTranslation('alerts');
	const { isLoading, isRefetching, isError, data, isValidRuleId, ruleId } =
		useGetAlertRuleDetailsStats();

	useEffect(() => {
		if (data?.payload?.data?.totalCurrentTriggers !== undefined) {
			setTotalCurrentTriggers(data.payload.data.totalCurrentTriggers);
		}
	}, [data, setTotalCurrentTriggers]);

	return (
		<DataStateRenderer
			isLoading={isLoading}
			isRefetching={isRefetching}
			isError={isError || !isValidRuleId || !ruleId}
			data={data?.payload?.data || null}
		>
			{(data): JSX.Element => {
				const {
					currentAvgResolutionTime,
					pastAvgResolutionTime,
					totalCurrentTriggers,
					totalPastTriggers,
					currentAvgResolutionTimeSeries,
					currentTriggersSeries,
				} = data;

				return (
					<>
						{hasTotalTriggeredStats(totalCurrentTriggers, totalPastTriggers) ? (
							<TotalTriggeredCard
								totalCurrentTriggers={totalCurrentTriggers}
								totalPastTriggers={totalPastTriggers}
								timeSeries={currentTriggersSeries?.values}
							/>
						) : (
							<StatsCard
								title={t('hist_total_triggered')}
								isEmpty
								emptyMessage={t('hist_none_triggered')}
							/>
						)}

						{hasAvgResolutionTimeStats(
							currentAvgResolutionTime,
							pastAvgResolutionTime,
						) ? (
							<AverageResolutionCard
								currentAvgResolutionTime={currentAvgResolutionTime}
								pastAvgResolutionTime={pastAvgResolutionTime}
								timeSeries={currentAvgResolutionTimeSeries?.values}
							/>
						) : (
							<StatsCard
								title={t('hist_avg_resolution_time')}
								isEmpty
								emptyMessage={t('hist_no_resolutions')}
							/>
						)}
					</>
				);
			}}
		</DataStateRenderer>
	);
}

export default StatsCardsRenderer;
