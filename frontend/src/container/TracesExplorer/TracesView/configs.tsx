import { generatePath, Link } from 'react-router-dom';
import type { TableColumnsType as ColumnsType } from 'antd';
import { Typography } from 'antd';
import ROUTES from 'constants/routes';
import { getMs } from 'container/Trace/Filters/Panel/PanelBody/Duration/util';
import { DEFAULT_PER_PAGE_OPTIONS } from 'hooks/queryPagination';
import { TFunction } from 'i18next';
import { ListItem } from 'types/api/widgets/getQuery';

export const PER_PAGE_OPTIONS: number[] = [10, ...DEFAULT_PER_PAGE_OPTIONS];

export const getColumns = (t: TFunction): ColumnsType<ListItem['data']> => [
	{
		title: t('col_root_service_name').toString(),
		dataIndex: 'service.name',
		key: 'serviceName',
	},
	{
		title: t('col_root_operation_name').toString(),
		dataIndex: 'name',
		key: 'name',
	},
	{
		title: t('col_root_duration_ms').toString(),
		dataIndex: 'duration_nano',
		key: 'durationNano',
		render: (duration: number): JSX.Element => (
			<Typography>{getMs(String(duration))}ms</Typography>
		),
	},
	{
		title: t('col_no_of_spans').toString(),
		dataIndex: 'span_count',
		key: 'span_count',
	},
	{
		title: t('col_trace_id').toString(),
		dataIndex: 'trace_id',
		key: 'traceID',
		render: (traceID: string): JSX.Element => (
			<Link
				to={generatePath(ROUTES.TRACE_DETAIL, {
					id: traceID,
				})}
				data-testid="trace-id"
			>
				{traceID}
			</Link>
		),
	},
];
