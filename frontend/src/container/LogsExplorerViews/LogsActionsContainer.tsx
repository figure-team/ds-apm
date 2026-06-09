import { useTranslation } from 'react-i18next';
import { Switch, Typography } from 'antd';
import DownloadOptionsMenu from 'components/DownloadOptionsMenu/DownloadOptionsMenu';
import LogsFormatOptionsMenu from 'components/LogsFormatOptionsMenu/LogsFormatOptionsMenu';
import ListViewOrderBy from 'components/OrderBy/ListViewOrderBy';
import { LOCALSTORAGE } from 'constants/localStorage';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { useOptionsMenu } from 'container/OptionsMenu';
import { ArrowUp10, Minus } from 'lucide-react';
import { DataSource, StringOperators } from 'types/common/queryBuilder';

import QueryStatus from './QueryStatus';

function LogsActionsContainer({
	listQuery,
	selectedPanelType,
	showFrequencyChart,
	handleToggleFrequencyChart,
	orderBy,
	setOrderBy,
	isFetching,
	isLoading,
	isError,
	isSuccess,
}: {
	listQuery: any;
	selectedPanelType: PANEL_TYPES;
	showFrequencyChart: boolean;
	handleToggleFrequencyChart: () => void;
	orderBy: string;
	setOrderBy: (value: string) => void;
	isFetching: boolean;
	isLoading: boolean;
	isError: boolean;
	isSuccess: boolean;
}): JSX.Element {
	const { t } = useTranslation(['logs']);
	const { options, config } = useOptionsMenu({
		storageKey: LOCALSTORAGE.LOGS_LIST_OPTIONS,
		dataSource: DataSource.LOGS,
		aggregateOperator: listQuery?.aggregateOperator || StringOperators.NOOP,
	});

	const formatItems = [
		{
			key: 'raw',
			label: t('logs:raw'),
			data: {
				title: t('logs:max_lines_per_row'),
			},
		},
		{
			key: 'list',
			label: t('logs:default'),
		},
		{
			key: 'table',
			label: t('logs:column'),
			data: {
				title: t('logs:columns'),
			},
		},
	];

	return (
		<div className="logs-actions-container">
			<div className="tab-options">
				<div className="tab-options-left">
					{selectedPanelType === PANEL_TYPES.LIST && (
						<div className="frequency-chart-view-controller">
							<Typography>{t('logs:frequency_chart')}</Typography>
							<Switch
								size="small"
								checked={showFrequencyChart}
								defaultChecked
								onChange={handleToggleFrequencyChart}
							/>
						</div>
					)}
				</div>

				<div className="tab-options-right">
					{selectedPanelType === PANEL_TYPES.LIST && (
						<>
							<div className="order-by-container">
								<div className="order-by-label">
									{t('logs:order_by')} <Minus size={14} /> <ArrowUp10 size={14} />
								</div>

								<ListViewOrderBy
									value={orderBy}
									onChange={(value): void => setOrderBy(value)}
									dataSource={DataSource.LOGS}
								/>
							</div>
							<div className="download-options-container">
								<DownloadOptionsMenu
									dataSource={DataSource.LOGS}
									selectedColumns={options?.selectColumns}
								/>
							</div>
							<div className="format-options-container">
								<LogsFormatOptionsMenu
									items={formatItems}
									selectedOptionFormat={options.format}
									config={config}
								/>
							</div>
						</>
					)}

					{(selectedPanelType === PANEL_TYPES.TIME_SERIES ||
						selectedPanelType === PANEL_TYPES.TABLE) && (
						<div className="query-stats">
							<QueryStatus
								loading={isLoading || isFetching}
								error={isError}
								success={isSuccess}
							/>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

export default LogsActionsContainer;
