import { useTranslation } from 'react-i18next';
import { Empty, Typography } from 'antd';

interface EmptyMetricsSearchProps {
	hasQueryResult?: boolean;
}

export default function EmptyMetricsSearch({
	hasQueryResult,
}: EmptyMetricsSearchProps): JSX.Element {
	const { t } = useTranslation(['metricsExplorer', 'common']);
	return (
		<div className="empty-metrics-search">
			<Empty
				description={
					<Typography.Title level={5}>
						{hasQueryResult
							? t('common:no_data')
							: t('metricsExplorer:empty_select_metric')}
					</Typography.Title>
				}
			/>
		</div>
	);
}
