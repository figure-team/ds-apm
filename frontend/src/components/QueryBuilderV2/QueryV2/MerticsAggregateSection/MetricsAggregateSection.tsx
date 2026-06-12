import { memo, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Tooltip } from 'antd';
import cx from 'classnames';
import InputWithLabel from 'components/InputWithLabel/InputWithLabel';
import { ATTRIBUTE_TYPES, PANEL_TYPES } from 'constants/queryBuilder';
import SpaceAggregationOptions from 'container/QueryBuilder/components/SpaceAggregationOptions/SpaceAggregationOptions';
import { GroupByFilter, OperatorsSelect } from 'container/QueryBuilder/filters';
import { useQueryOperations } from 'hooks/queryBuilder/useQueryBuilderOperations';
import { IBuilderQuery } from 'types/api/queryBuilder/queryBuilderData';
import { MetricAggregation } from 'types/api/v5/queryRange';

import { useQueryBuilderV2Context } from '../../QueryBuilderV2Context';

import './MetricsAggregateSection.styles.scss';

const MetricsAggregateSection = memo(function MetricsAggregateSection({
	query,
	index,
	version,
	panelType,
	signalSource = '',
}: {
	query: IBuilderQuery;
	index: number;
	version: string;
	panelType: PANEL_TYPES | null;
	signalSource: string;
}): JSX.Element {
	const { t } = useTranslation('common');
	const { setAggregationOptions } = useQueryBuilderV2Context();
	const {
		operators,
		spaceAggregationOptions,
		handleChangeQueryData,
		handleChangeOperator,
		handleSpaceAggregationChange,
	} = useQueryOperations({
		index,
		query,
		entityVersion: version,
	});

	// this function is only relevant for metrics and now operators are part of aggregations
	const queryAggregation = useMemo(
		() => query.aggregations?.[0] as MetricAggregation,
		[query.aggregations],
	);

	const isHistogram = useMemo(
		() => query.aggregateAttribute?.type === ATTRIBUTE_TYPES.HISTOGRAM,
		[query.aggregateAttribute?.type],
	);

	useEffect(() => {
		setAggregationOptions(query.queryName, [
			{
				func: queryAggregation.spaceAggregation || 'count',
				arg: queryAggregation.metricName || '',
			},
		]);
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [
		queryAggregation.spaceAggregation,
		queryAggregation.metricName,
		query.queryName,
	]);

	const handleChangeGroupByKeys = useCallback(
		(value: IBuilderQuery['groupBy']) => {
			handleChangeQueryData('groupBy', value);
		},
		[handleChangeQueryData],
	);

	const handleChangeAggregateEvery = useCallback(
		(value: string) => {
			handleChangeQueryData('stepInterval', Number(value));
		},
		[handleChangeQueryData],
	);

	const showAggregationInterval = useMemo(
		() => panelType !== PANEL_TYPES.VALUE,
		[panelType],
	);

	const disableOperatorSelector =
		!queryAggregation.metricName || queryAggregation.metricName === '';

	return (
		<div
			className={cx('metrics-aggregate-section', {
				'is-histogram': isHistogram,
			})}
		>
			{!isHistogram && (
				<div className="non-histogram-container">
					<div className="metrics-time-aggregation-section">
						<div className="metrics-aggregation-section-content">
							<div className="metrics-aggregation-section-content-item">
								<Tooltip
									title={
										<a
											href="https://signoz.io/docs/metrics-management/types-and-aggregation/#aggregation"
											target="_blank"
											rel="noopener noreferrer"
											style={{ color: '#1890ff', textDecoration: 'underline' }}
										>
											{t('query_builder.learn_temporal_aggregation')}
										</a>
									}
								>
									<div className="metrics-aggregation-section-content-item-label main-label">
										{t('query_builder.aggregate_within_time_series')}{' '}
									</div>
								</Tooltip>
								<div className="metrics-aggregation-section-content-item-value">
									<OperatorsSelect
										value={queryAggregation.timeAggregation || ''}
										onChange={handleChangeOperator}
										operators={operators}
										className="metrics-operators-select"
									/>
								</div>
							</div>

							{showAggregationInterval && (
								<div className="metrics-aggregation-section-content-item">
									<Tooltip
										title={
											<div>
												{t('query_builder.set_aggregation_interval')}
												<br />
												<a
													href="https://signoz.io/docs/userguide/query-builder-v5/#time-aggregation-windows"
													target="_blank"
													rel="noopener noreferrer"
													style={{ color: '#1890ff', textDecoration: 'underline' }}
												>
													{t('query_builder.learn_step_intervals')}
												</a>
											</div>
										}
										placement="top"
									>
										<div
											className="metrics-aggregation-section-content-item-label"
											style={{ cursor: 'help' }}
										>
											{t('query_builder.every')}
										</div>
									</Tooltip>

									<div className="metrics-aggregation-section-content-item-value">
										<InputWithLabel
											onChange={handleChangeAggregateEvery}
											label={t('query_builder.seconds')}
											placeholder={t('query_builder.auto')}
											labelAfter
											initialValue={query?.stepInterval ?? null}
											inputId="Seconds"
										/>
									</div>
								</div>
							)}
						</div>
					</div>
					<div className="metrics-space-aggregation-section">
						<div className="metrics-aggregation-section-content">
							<div className="metrics-aggregation-section-content-item">
								<Tooltip
									title={
										<a
											href="https://signoz.io/docs/metrics-management/types-and-aggregation/#aggregation"
											target="_blank"
											rel="noopener noreferrer"
											style={{ color: '#1890ff', textDecoration: 'underline' }}
										>
											{t('query_builder.learn_spatial_aggregation')}
										</a>
									}
								>
									<div className="metrics-aggregation-section-content-item-label main-label">
										{t('query_builder.aggregate_across_time_series')}
									</div>
								</Tooltip>
								<div className="metrics-aggregation-section-content-item-value">
									<SpaceAggregationOptions
										panelType={panelType}
										key={`${panelType}${queryAggregation.spaceAggregation}${queryAggregation.timeAggregation}`}
										aggregatorAttributeType={
											query?.aggregateAttribute?.type as ATTRIBUTE_TYPES
										}
										selectedValue={queryAggregation.spaceAggregation || ''}
										disabled={disableOperatorSelector}
										onSelect={handleSpaceAggregationChange}
										operators={spaceAggregationOptions}
										qbVersion="v3"
									/>
								</div>
							</div>

							<div className="metrics-aggregation-section-content-item">
								<div className="metrics-aggregation-section-content-item-label">
									{t('query_builder.by')}
								</div>

								<div className="metrics-aggregation-section-content-item-value group-by-filter-container">
									<GroupByFilter
										disabled={!queryAggregation.metricName}
										query={query}
										onChange={handleChangeGroupByKeys}
										signalSource={signalSource}
									/>
								</div>
							</div>
						</div>
					</div>
				</div>
			)}

			{isHistogram && (
				<div className="metrics-space-aggregation-section">
					<div className="metrics-aggregation-section-content">
						<div className="metrics-aggregation-section-content-item">
							<div className="metrics-aggregation-section-content-item-value">
								<SpaceAggregationOptions
									panelType={panelType}
									key={`${panelType}${queryAggregation.spaceAggregation}${queryAggregation.timeAggregation}`}
									aggregatorAttributeType={
										query?.aggregateAttribute?.type as ATTRIBUTE_TYPES
									}
									selectedValue={queryAggregation.spaceAggregation || ''}
									disabled={disableOperatorSelector}
									onSelect={handleSpaceAggregationChange}
									operators={spaceAggregationOptions}
									qbVersion="v3"
								/>
							</div>
						</div>

						<div className="metrics-aggregation-section-content-item">
							<div className="metrics-aggregation-section-content-item-label">
									{t('query_builder.by')}
								</div>

							<div className="metrics-aggregation-section-content-item-value group-by-filter-container">
								<GroupByFilter
									disabled={!queryAggregation.metricName}
									query={query}
									onChange={handleChangeGroupByKeys}
									signalSource={signalSource}
								/>
							</div>
						</div>
						<div className="metrics-aggregation-section-content-item">
							<Tooltip
								title={
									<div>
										{t('query_builder.set_aggregation_interval')}
										<br />
										<a
											href="https://signoz.io/docs/userguide/query-builder-v5/#time-aggregation-windows"
											target="_blank"
											rel="noopener noreferrer"
											style={{ color: '#1890ff', textDecoration: 'underline' }}
										>
											{t('query_builder.learn_step_intervals')}
										</a>
									</div>
								}
								placement="top"
							>
								<div
									className="metrics-aggregation-section-content-item-label"
									style={{ cursor: 'help' }}
								>
									{t('query_builder.every')}
								</div>
							</Tooltip>

							<div className="metrics-aggregation-section-content-item-value">
								<InputWithLabel
									onChange={handleChangeAggregateEvery}
									label={t('query_builder.seconds')}
									placeholder={t('query_builder.auto')}
									labelAfter
									initialValue={query?.stepInterval ?? null}
									className="histogram-every-input"
									inputId="Seconds"
								/>
							</div>
						</div>
					</div>
				</div>
			)}
		</div>
	);
});

export default MetricsAggregateSection;
