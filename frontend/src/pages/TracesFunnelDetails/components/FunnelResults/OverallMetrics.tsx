import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useFunnelMetrics } from 'hooks/TracesFunnels/useFunnelMetrics';

import FunnelMetricsTable from './FunnelMetricsTable';

function OverallMetrics(): JSX.Element {
	const { t } = useTranslation('trace');
	const { funnelId } = useParams<{ funnelId: string }>();
	const { isLoading, metricsData, conversionRate, isError } = useFunnelMetrics({
		funnelId,
	});

	return (
		<FunnelMetricsTable
			title={t('funnels.overall_metrics_title')}
			subtitle={{
				label: t('funnels.conversion_rate'),
				value: `${conversionRate.toFixed(2)}%`,
			}}
			isLoading={isLoading}
			isError={isError}
			data={metricsData}
		/>
	);
}

export default OverallMetrics;
