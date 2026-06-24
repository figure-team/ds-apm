import { ReactNode, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import { Space, Typography } from 'antd';
import TextToolTip from 'components/TextToolTip';
import { apDexToolTipUrl } from 'constants/apDex';
import { ENTITY_VERSION_V4 } from 'constants/app';
import { PANEL_TYPES } from 'constants/queryBuilder';
import Graph from 'container/GridCardLayout/GridCard';
import DisplayThreshold from 'container/GridCardLayout/WidgetHeader/DisplayThreshold';
import { SERVICE_CHART_ID } from 'container/MetricsApplication/constant';
import { getWidgetQueryBuilder } from 'container/MetricsApplication/MetricsApplication.factory';
import { apDexMetricsQueryBuilderQueries } from 'container/MetricsApplication/MetricsPageQueries/OverviewQueries';
import { EQueryType } from 'types/common/dashboard';
import { v4 as uuid } from 'uuid';

import { FeatureKeys } from '../../../../../constants/features';
import { useAppContext } from '../../../../../providers/App/App';
import { IServiceName } from '../../types';
import { ApDexMetricsProps } from './types';

function ApDexMetrics({
	delta,
	metricsBuckets,
	thresholdValue,
	onDragSelect,
	tagFilterItems,
	topLevelOperationsRoute,
	handleGraphClick,
}: ApDexMetricsProps): JSX.Element {
	const { t } = useTranslation(['services']);
	const { servicename: encodedServiceName } = useParams<IServiceName>();
	const servicename = decodeURIComponent(encodedServiceName);
	const { featureFlags } = useAppContext();
	const dotMetricsEnabled =
		featureFlags?.find((flag) => flag.name === FeatureKeys.DOT_METRICS_ENABLED)
			?.active || false;
	const apDexMetricsWidget = useMemo(
		() =>
			getWidgetQueryBuilder({
				query: {
					queryType: EQueryType.QUERY_BUILDER,
					promql: [],
					builder: apDexMetricsQueryBuilderQueries({
						servicename,
						tagFilterItems,
						topLevelOperationsRoute,
						threashold: thresholdValue || 0,
						delta: delta || false,
						metricsBuckets: metricsBuckets || [],
						dotMetricsEnabled,
					}),
					clickhouse_sql: [],
					id: uuid(),
				},
				title: (
					<Space>
						<Typography>{t('services:graph_apdex')}</Typography>
						<TextToolTip
							text={t('services:apdex_tooltip')}
							url={apDexToolTipUrl}
							useFilledIcon={false}
							urlText={t('services:apdex_learn_more')}
						/>
					</Space>
				),
				panelTypes: PANEL_TYPES.TIME_SERIES,
				id: SERVICE_CHART_ID.apdex,
			}),
		[
			delta,
			metricsBuckets,
			servicename,
			tagFilterItems,
			thresholdValue,
			topLevelOperationsRoute,
			dotMetricsEnabled,
			t,
		],
	);

	const threshold: ReactNode = useMemo(() => {
		if (thresholdValue) {
			return <DisplayThreshold threshold={thresholdValue} />;
		}
		return null;
	}, [thresholdValue]);

	const isQueryEnabled =
		topLevelOperationsRoute.length > 0 &&
		!!metricsBuckets &&
		metricsBuckets?.length > 0 &&
		delta !== undefined;

	return (
		<Graph
			widget={apDexMetricsWidget}
			onDragSelect={onDragSelect}
			onClickHandler={handleGraphClick('ApDex')}
			threshold={threshold}
			isQueryEnabled={isQueryEnabled}
			version={ENTITY_VERSION_V4}
		/>
	);
}

ApDexMetrics.defaultProps = {
	delta: undefined,
	le: undefined,
};

export default ApDexMetrics;
