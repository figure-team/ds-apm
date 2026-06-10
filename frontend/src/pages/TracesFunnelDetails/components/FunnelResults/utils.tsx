import { TableColumnProps } from 'antd';
import { TFunction } from 'i18next';
import { Link } from 'react-router-dom';
import { getYAxisFormattedValue } from 'components/Graph/yAxisConfig';

interface TopTraceRow {
	trace_id: string;
	duration_ms: string;
}

export const getTopTracesTableColumns = (
	t: TFunction,
): Array<TableColumnProps<TopTraceRow>> => [
	{
		title: t('funnels.col_trace_id').toString(),
		dataIndex: 'trace_id',
		key: 'trace_id',
		render: (traceId: string): JSX.Element => (
			<Link to={`/trace/${traceId}`} className="trace-id-cell">
				{traceId}
			</Link>
		),
	},
	{
		title: t('funnels.col_step_transition_duration').toString(),
		dataIndex: 'duration_ms',
		key: 'duration_ms',
		render: (value: string): string => getYAxisFormattedValue(`${value}`, 'ms'),
	},
];
