import { FilterOutlined, VerticalAlignTopOutlined } from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import cx from 'classnames';
import { Atom, Binoculars, SquareMousePointer, Terminal } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ExplorerViews } from 'pages/LogsExplorer/utils';

import './ToolbarActions.styles.scss';

interface LeftToolbarActionsProps {
	items: any;
	selectedView: string;
	onChangeSelectedView: (view: ExplorerViews) => void;
	showFilter: boolean;
	handleFilterVisibilityChange: () => void;
}

const activeTab = 'active-tab';

export default function LeftToolbarActions({
	items,
	selectedView,
	onChangeSelectedView,
	showFilter,
	handleFilterVisibilityChange,
}: LeftToolbarActionsProps): JSX.Element {
	const { t } = useTranslation('common');
	const { clickhouse, list, timeseries, table, trace } = items;

	return (
		<div className="left-toolbar">
			{!showFilter && (
				<Tooltip title={t('explorer.show_filters')}>
					<Button onClick={handleFilterVisibilityChange} className="filter-btn">
						<FilterOutlined />
						<VerticalAlignTopOutlined rotate={90} />
					</Button>
				</Tooltip>
			)}
			<div className="left-toolbar-query-actions">
				{list?.show && (
					<Tooltip title={t('explorer.view_list')}>
						<Button
							disabled={list.disabled}
							className={cx(
								'list-view-tab',
								'explorer-view-option',
								selectedView === list.key ? activeTab : '',
							)}
							onClick={(): void => onChangeSelectedView(list.key)}
						>
							<SquareMousePointer size={14} data-testid="search-view" />
							{t('explorer.view_list')}
						</Button>
					</Tooltip>
				)}

				{trace?.show && (
					<Tooltip title={t('explorer.view_trace')}>
						<Button
							disabled={trace.disabled}
							className={cx(
								'trace-view-tab',
								'explorer-view-option',
								selectedView === trace.key ? activeTab : '',
							)}
							onClick={(): void => onChangeSelectedView(trace.key)}
						>
							<SquareMousePointer size={14} data-testid="trace-view" />
							{t('explorer.view_trace')}
						</Button>
					</Tooltip>
				)}

				{timeseries?.show && (
					<Tooltip title={t('explorer.view_timeseries')}>
						<Button
							disabled={timeseries.disabled}
							className={cx(
								'timeseries-view-tab',
								'explorer-view-option',
								selectedView === timeseries.key ? activeTab : '',
							)}
							onClick={(): void => onChangeSelectedView(timeseries.key)}
						>
							<Atom size={14} data-testid="query-builder-view" />
							{t('explorer.view_timeseries')}
						</Button>
					</Tooltip>
				)}

				{clickhouse?.show && (
					<Tooltip title={t('explorer.view_clickhouse')}>
						<Button
							disabled={clickhouse.disabled}
							className={cx(
								'clickhouse-view-tab',
								'explorer-view-option',
								selectedView === clickhouse.key ? activeTab : '',
							)}
							onClick={(): void => onChangeSelectedView(clickhouse.key)}
						>
							<Terminal size={14} data-testid="clickhouse-view" />
							{t('explorer.view_clickhouse')}
						</Button>
					</Tooltip>
				)}

				{table?.show && (
					<Tooltip title={t('explorer.view_table')}>
						<Button
							disabled={table.disabled}
							className={cx(
								'table-view-tab',
								'explorer-view-option',
								selectedView === table.key ? activeTab : '',
							)}
							onClick={(): void => onChangeSelectedView(table.key)}
						>
							<Binoculars size={14} data-testid="query-builder-view-v2" />
							{t('explorer.view_table')}
						</Button>
					</Tooltip>
				)}
			</div>
		</div>
	);
}
