import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import QuerySearch from 'components/QueryBuilderV2/QueryV2/QuerySearch/QuerySearch';
import { convertExpressionToFilters } from 'components/QueryBuilderV2/utils';
import { DataSource } from 'types/common/queryBuilder';

import { MetricsExplorerEventKeys, MetricsExplorerEvents } from '../events';
import { MetricFiltersProps } from './types';

function MetricFilters({
	dispatchMetricInspectionOptions,
	currentQuery,
	setCurrentQuery,
}: MetricFiltersProps): JSX.Element {
	const { t } = useTranslation('metricsExplorer');
	const handleOnChange = useCallback(
		(expression: string): void => {
			logEvent(MetricsExplorerEvents.FilterApplied, {
				[MetricsExplorerEventKeys.Modal]: 'inspect',
			});
			const tagFilter = {
				items: convertExpressionToFilters(expression),
				op: 'AND',
			};
			setCurrentQuery({
				...currentQuery,
				filters: tagFilter,
				filter: {
					...currentQuery.filter,
					expression,
				},
				expression,
			});
			dispatchMetricInspectionOptions({
				type: 'SET_FILTERS',
				payload: expression,
			});
		},
		[currentQuery, dispatchMetricInspectionOptions, setCurrentQuery],
	);

	return (
		<div
			data-testid="metric-filters"
			className="inspect-metrics-input-group metric-filters"
		>
			<Typography.Text>{t('where')}</Typography.Text>
			<QuerySearch
				queryData={currentQuery}
				onChange={handleOnChange}
				dataSource={DataSource.METRICS}
			/>
		</div>
	);
}

export default MetricFilters;
