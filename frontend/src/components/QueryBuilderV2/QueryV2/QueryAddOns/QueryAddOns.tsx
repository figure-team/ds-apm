import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Radio, RadioChangeEvent, Tooltip } from 'antd';
import InputWithLabel from 'components/InputWithLabel/InputWithLabel';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { GroupByFilter } from 'container/QueryBuilder/filters/GroupByFilter/GroupByFilter';
import { OrderByFilter } from 'container/QueryBuilder/filters/OrderByFilter/OrderByFilter';
import { ReduceToFilter } from 'container/QueryBuilder/filters/ReduceToFilter/ReduceToFilter';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useQueryOperations } from 'hooks/queryBuilder/useQueryBuilderOperations';
import { get, isEmpty } from 'lodash-es';
import { BarChart2, ChevronUp, ExternalLink, ScrollText } from 'lucide-react';
import { IBuilderQuery } from 'types/api/queryBuilder/queryBuilderData';
import { MetricAggregation } from 'types/api/v5/queryRange';
import { DataSource, ReduceOperators } from 'types/common/queryBuilder';

import HavingFilter from './HavingFilter/HavingFilter';
import { buildDefaultLegendFromGroupBy } from './utils';

import './QueryAddOns.styles.scss';

interface AddOn {
	icon: React.ReactNode;
	label: string;
	key: string;
	description?: string;
	docLink?: string;
}

const ADD_ONS_KEYS = {
	GROUP_BY: 'group_by',
	HAVING: 'having',
	ORDER_BY: 'order_by',
	LIMIT: 'limit',
	LEGEND_FORMAT: 'legend_format',
	REDUCE_TO: 'reduce_to',
};

const ADD_ONS_KEYS_TO_QUERY_PATH = {
	[ADD_ONS_KEYS.GROUP_BY]: 'groupBy',
	[ADD_ONS_KEYS.HAVING]: 'having.expression',
	[ADD_ONS_KEYS.ORDER_BY]: 'orderBy',
	[ADD_ONS_KEYS.LIMIT]: 'limit',
	[ADD_ONS_KEYS.LEGEND_FORMAT]: 'legend',
	[ADD_ONS_KEYS.REDUCE_TO]: 'reduceTo',
};

// `label` holds the i18n key (common namespace); resolved with t() at render time.
const ADD_ONS = [
	{
		icon: <BarChart2 size={14} />,
		label: 'query_builder.group_by',
		key: ADD_ONS_KEYS.GROUP_BY,
		description: 'query_builder.group_by_desc',
		docLink: 'https://signoz.io/docs/userguide/query-builder-v5/#grouping',
	},
	{
		icon: <ScrollText size={14} />,
		label: 'query_builder.having',
		key: ADD_ONS_KEYS.HAVING,
		description: 'query_builder.having_desc',
		docLink:
			'https://signoz.io/docs/userguide/query-builder-v5/#conditional-filtering-with-having',
	},
	{
		icon: <ScrollText size={14} />,
		label: 'query_builder.order_by',
		key: ADD_ONS_KEYS.ORDER_BY,
		description: 'query_builder.order_by_desc',
		docLink:
			'https://signoz.io/docs/userguide/query-builder-v5/#sorting--limiting',
	},
	{
		icon: <ScrollText size={14} />,
		label: 'query_builder.limit',
		key: ADD_ONS_KEYS.LIMIT,
		description: 'query_builder.limit_desc',
		docLink:
			'https://signoz.io/docs/userguide/query-builder-v5/#sorting--limiting',
	},
	{
		icon: <ScrollText size={14} />,
		label: 'query_builder.legend_format',
		key: ADD_ONS_KEYS.LEGEND_FORMAT,
		description: 'query_builder.legend_format_desc',
		docLink:
			'https://signoz.io/docs/userguide/query-builder-v5/#legend-formatting',
	},
];

const REDUCE_TO = {
	icon: <ScrollText size={14} />,
	label: 'query_builder.reduce_to',
	key: ADD_ONS_KEYS.REDUCE_TO,
	description: 'query_builder.reduce_to_desc',
	docLink:
		'https://signoz.io/docs/userguide/query-builder-v5/#reduce-operations',
};

const hasValue = (value: unknown): boolean =>
	value != null && value !== '' && !(Array.isArray(value) && value.length === 0);

// Custom tooltip content component
function TooltipContent({
	label,
	description,
	docLink,
}: {
	label: string;
	description?: string;
	docLink?: string;
}): JSX.Element {
	const { t } = useTranslation('common');
	return (
		<div
			style={{
				display: 'flex',
				flexDirection: 'column',
				gap: '8px',
				maxWidth: '300px',
			}}
		>
			<strong style={{ fontSize: '14px' }}>{label}</strong>
			{description && (
				<span style={{ fontSize: '12px', lineHeight: '1.5' }}>{description}</span>
			)}
			{docLink && (
				<a
					href={docLink}
					target="_blank"
					rel="noopener noreferrer"
					onClick={(e): void => e.stopPropagation()}
					style={{
						display: 'flex',
						alignItems: 'center',
						gap: '4px',
						color: '#4096ff',
						fontSize: '12px',
						marginTop: '4px',
					}}
				>
					{t('learn_more')}
					<ExternalLink size={12} />
				</a>
			)}
		</div>
	);
}

function QueryAddOns({
	query,
	version,
	isListViewPanel,
	showReduceTo,
	panelType,
	index,
	isForTraceOperator = false,
}: {
	query: IBuilderQuery;
	version: string;
	isListViewPanel: boolean;
	showReduceTo: boolean;
	panelType: PANEL_TYPES | null;
	index: number;
	isForTraceOperator?: boolean;
}): JSX.Element {
	const { t } = useTranslation('common');
	const [addOns, setAddOns] = useState<AddOn[]>(ADD_ONS);

	const [selectedViews, setSelectedViews] = useState<AddOn[]>([]);

	const initializedRef = useRef(false);
	const prevAvailableKeysRef = useRef<Set<string> | null>(null);

	const { handleChangeQueryData } = useQueryOperations({
		index,
		query,
		entityVersion: '',
		isForTraceOperator,
	});

	const { handleSetQueryData } = useQueryBuilder();

	useEffect(() => {
		if (isListViewPanel) {
			setAddOns([]);

			setSelectedViews([
				ADD_ONS.find((addOn) => addOn.key === ADD_ONS_KEYS.ORDER_BY) as AddOn,
			]);

			return;
		}

		let filteredAddOns: AddOn[];
		if (panelType === PANEL_TYPES.VALUE) {
			// Filter out all add-ons except legend format
			filteredAddOns = ADD_ONS.filter(
				(addOn) => addOn.key === ADD_ONS_KEYS.LEGEND_FORMAT,
			);
		} else {
			filteredAddOns = Object.values(ADD_ONS);

			if (query.dataSource === DataSource.METRICS) {
				// Filter out group_by for metrics data source (handled in MetricsAggregateSection)
				filteredAddOns = filteredAddOns.filter(
					(addOn) => addOn.key !== ADD_ONS_KEYS.GROUP_BY,
				);
			}
		}

		if (showReduceTo) {
			filteredAddOns = [...filteredAddOns, REDUCE_TO];
		}
		setAddOns(filteredAddOns);

		const availableAddOnKeys = new Set(filteredAddOns.map((a) => a.key));
		const previousKeys = prevAvailableKeysRef.current;
		const hasAvailabilityItemsChanged =
			previousKeys !== null &&
			(previousKeys.size !== availableAddOnKeys.size ||
				[...availableAddOnKeys].some((key) => !previousKeys.has(key)));
		prevAvailableKeysRef.current = availableAddOnKeys;

		if (!initializedRef.current || hasAvailabilityItemsChanged) {
			initializedRef.current = true;

			const activeAddOnKeys = new Set(
				Object.entries(ADD_ONS_KEYS_TO_QUERY_PATH)
					.filter(([, path]) => hasValue(get(query, path)))
					.map(([key]) => key),
			);

			// Initial seeding from query values on mount
			setSelectedViews(
				filteredAddOns.filter(
					(addOn) =>
						activeAddOnKeys.has(addOn.key) && availableAddOnKeys.has(addOn.key),
				),
			);
			return;
		}

		setSelectedViews((prev) =>
			prev.filter((view) =>
				filteredAddOns.some((addOn) => addOn.key === view.key),
			),
		);
	}, [panelType, isListViewPanel, query, showReduceTo]);

	const handleOptionClick = (e: RadioChangeEvent): void => {
		const clickedAddOn = e.target.value as AddOn;
		const isAlreadySelected = selectedViews.some(
			(view) => view.key === clickedAddOn.key,
		);

		if (isAlreadySelected) {
			setSelectedViews((prev) =>
				prev.filter((view) => view.key !== clickedAddOn.key),
			);
		} else {
			// When enabling Legend format for the first time with an empty legend
			// and existing group-by keys, prefill the legend using all group-by keys.
			// This keeps existing custom legends intact and only helps seed a sensible default.
			if (
				clickedAddOn.key === ADD_ONS_KEYS.LEGEND_FORMAT &&
				isEmpty(query?.legend) &&
				Array.isArray(query.groupBy) &&
				query.groupBy.length > 0
			) {
				const defaultLegend = buildDefaultLegendFromGroupBy(query.groupBy);

				if (defaultLegend) {
					handleChangeQueryLegend(defaultLegend);
				}
			}

			setSelectedViews((prev) => [...prev, clickedAddOn]);
		}
	};

	const handleChangeGroupByKeys = useCallback(
		(value: IBuilderQuery['groupBy']) => {
			handleChangeQueryData('groupBy', value);
		},
		[handleChangeQueryData],
	);

	const handleChangeOrderByKeys = useCallback(
		(value: IBuilderQuery['orderBy']) => {
			handleChangeQueryData('orderBy', value);
		},
		[handleChangeQueryData],
	);

	const handleChangeReduceToV5 = useCallback(
		(value: ReduceOperators) => {
			handleSetQueryData(index, {
				...query,
				aggregations: [
					{
						...(query.aggregations?.[0] as MetricAggregation),
						reduceTo: value,
					},
				],
			});
		},
		[handleSetQueryData, index, query],
	);

	const handleRemoveView = useCallback((key: string): void => {
		setSelectedViews((prev) => prev.filter((view) => view.key !== key));
	}, []);

	const handleChangeQueryLegend = useCallback(
		(value: string) => {
			handleChangeQueryData('legend', value);
		},
		[handleChangeQueryData],
	);

	const handleChangeLimit = useCallback(
		(value: string) => {
			handleChangeQueryData('limit', Number(value) || null);
		},
		[handleChangeQueryData],
	);

	const handleChangeHaving = useCallback(
		(value: string) => {
			handleChangeQueryData('having', {
				expression: value,
			});
		},
		[handleChangeQueryData],
	);

	return (
		<div className="query-add-ons" data-testid="query-add-ons">
			{selectedViews.length > 0 && (
				<div className="selected-add-ons-content">
					{selectedViews.find((view) => view.key === 'group_by') && (
						<div className="add-on-content" data-testid="group-by-content">
							<div className="periscope-input-with-label">
								<Tooltip
									title={
										<TooltipContent
											label={t('query_builder.group_by')}
											description={t('query_builder.group_by_desc')}
											docLink="https://signoz.io/docs/userguide/query-builder-v5/#grouping"
										/>
									}
									placement="top"
									mouseEnterDelay={0.5}
								>
									<div className="label" style={{ cursor: 'help' }}>
										{t('query_builder.group_by')}
									</div>
								</Tooltip>
								<div className="input">
									<GroupByFilter
										disabled={
											query.dataSource === DataSource.METRICS &&
											!(query.aggregations?.[0] as MetricAggregation)?.metricName
										}
										query={query}
										onChange={handleChangeGroupByKeys}
									/>
								</div>
								<Button
									className="close-btn periscope-btn ghost"
									icon={<ChevronUp size={16} />}
									onClick={(): void => handleRemoveView('group_by')}
								/>
							</div>
						</div>
					)}
					{selectedViews.find((view) => view.key === 'having') && (
						<div className="add-on-content" data-testid="having-content">
							<div className="periscope-input-with-label">
								<Tooltip
									title={
										<TooltipContent
											label={t('query_builder.having')}
											description={t('query_builder.having_desc')}
											docLink="https://signoz.io/docs/userguide/query-builder-v5/#conditional-filtering-with-having"
										/>
									}
									placement="top"
									mouseEnterDelay={0.5}
								>
									<div className="label" style={{ cursor: 'help' }}>
										{t('query_builder.having')}
									</div>
								</Tooltip>
								<div className="input">
									<HavingFilter
										onClose={(): void => {
											setSelectedViews((prev) =>
												prev.filter((view) => view.key !== 'having'),
											);
										}}
										onChange={handleChangeHaving}
										queryData={query}
									/>
								</div>
							</div>
						</div>
					)}
					{selectedViews.find((view) => view.key === 'limit') && (
						<div className="add-on-content" data-testid="limit-content">
							<InputWithLabel
								label={t('query_builder.limit')}
								type="number"
								onChange={handleChangeLimit}
								initialValue={query?.limit ?? undefined}
								placeholder={t('query_builder.limit_placeholder')}
								onClose={(): void => {
									setSelectedViews((prev) =>
										prev.filter((view) => view.key !== 'limit'),
									);
								}}
								closeIcon={<ChevronUp size={16} />}
								inputId="Limit"
							/>
						</div>
					)}
					{selectedViews.find((view) => view.key === 'order_by') && (
						<div className="add-on-content" data-testid="order-by-content">
							<div className="periscope-input-with-label">
								<Tooltip
									title={
										<TooltipContent
											label={t('query_builder.order_by')}
											description={t('query_builder.order_by_desc')}
											docLink="https://signoz.io/docs/userguide/query-builder-v5/#sorting--limiting"
										/>
									}
									placement="top"
									mouseEnterDelay={0.5}
								>
									<div className="label" style={{ cursor: 'help' }}>
										{t('query_builder.order_by')}
									</div>
								</Tooltip>
								<div className="input">
									<OrderByFilter
										entityVersion={version}
										query={query}
										onChange={handleChangeOrderByKeys}
										isListViewPanel={isListViewPanel}
										isNewQueryV2
									/>
								</div>
								{!isListViewPanel && (
									<Button
										className="close-btn periscope-btn ghost"
										icon={<ChevronUp size={16} />}
										onClick={(): void => handleRemoveView('order_by')}
									/>
								)}
							</div>
						</div>
					)}

					{selectedViews.find((view) => view.key === 'reduce_to') &&
						showReduceTo && (
							<div className="add-on-content" data-testid="reduce-to-content">
								<div className="periscope-input-with-label">
									<Tooltip
										title={
											<TooltipContent
												label={t('query_builder.reduce_to')}
												description={t('query_builder.reduce_to_desc')}
												docLink="https://signoz.io/docs/userguide/query-builder-v5/#reduce-operations"
											/>
										}
										placement="top"
										mouseEnterDelay={0.5}
									>
										<div className="label" style={{ cursor: 'help' }}>
											{t('query_builder.reduce_to')}
										</div>
									</Tooltip>
									<div className="input">
										<ReduceToFilter query={query} onChange={handleChangeReduceToV5} />
									</div>

									<Button
										className="close-btn periscope-btn ghost"
										icon={<ChevronUp size={16} />}
										onClick={(): void => handleRemoveView('reduce_to')}
									/>
								</div>
							</div>
						)}

					{selectedViews.find((view) => view.key === 'legend_format') && (
						<div className="add-on-content" data-testid="legend-format-content">
							<InputWithLabel
								label={t('query_builder.legend_format')}
								placeholder={t('query_builder.legend_format_placeholder')}
								onChange={handleChangeQueryLegend}
								initialValue={isEmpty(query?.legend) ? undefined : query?.legend}
								onClose={(): void => {
									setSelectedViews((prev) =>
										prev.filter((view) => view.key !== 'legend_format'),
									);
								}}
								closeIcon={<ChevronUp size={16} />}
								inputId="Legend format"
							/>
						</div>
					)}
				</div>
			)}

			<div className="add-ons-list">
				<Radio.Group
					className="add-ons-tabs"
					onChange={handleOptionClick}
					value={selectedViews}
				>
					{addOns.map((addOn) => (
						<Tooltip
							key={addOn.key}
							title={
								<TooltipContent
									label={t(addOn.label)}
									description={
										addOn.description
											? t(addOn.description, {
													example: '{{service.name}}-{{endpoint}}',
											  })
											: undefined
									}
									docLink={addOn.docLink}
								/>
							}
							placement="top"
							mouseEnterDelay={0.5}
						>
							<Radio.Button
								className={
									selectedViews.find((view) => view.key === addOn.key)
										? 'selected-view tab'
										: 'tab'
								}
								value={addOn}
							>
								<div
									className="add-on-tab-title"
									data-testid={`query-add-on-${addOn.key}`}
								>
									{addOn.icon}
									{t(addOn.label)}
								</div>
							</Radio.Button>
						</Tooltip>
					))}
				</Radio.Group>
			</div>
		</div>
	);
}

export default QueryAddOns;
