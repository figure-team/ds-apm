import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { useMeasure } from 'react-use';
import { Typography } from 'antd';
import Graph from 'components/Graph';
import Spinner from 'components/Spinner';
import { AppState } from 'store/reducers';
import { TraceReducer } from 'types/reducer/trace';

import { getChartData, getChartDataforGroupBy } from './config';
import { Container } from './styles';

function TraceGraph(): JSX.Element {
	const { t } = useTranslation(['trace']);
	const [ref, { width }] = useMeasure();

	const { spansGraph, selectedGroupBy, yAxisUnit } = useSelector<
		AppState,
		TraceReducer
	>((state) => state.traces);

	const { loading, error, errorMessage, payload } = spansGraph;

	const ChartData = useMemo(
		() =>
			selectedGroupBy.length === 0 || selectedGroupBy === 'none'
				? getChartData(payload)
				: getChartDataforGroupBy(payload),
		[payload, selectedGroupBy],
	);

	if (error) {
		return (
			<Container center>
				<Typography>{errorMessage || t('something_went_wrong')}</Typography>
			</Container>
		);
	}

	if (loading) {
		return (
			<Container>
				<Spinner height="20vh" size="small" tip={t('loading')} />
			</Container>
		);
	}

	return (
		<Container ref={ref as never}>
			<Graph
				animate={false}
				data={ChartData}
				name="traceGraph"
				type="line"
				yAxisUnit={yAxisUnit}
				forceReRender={width}
			/>
		</Container>
	);
}

export default TraceGraph;
