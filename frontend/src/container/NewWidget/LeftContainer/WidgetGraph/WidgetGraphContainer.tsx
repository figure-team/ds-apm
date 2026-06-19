import { useTranslation } from 'react-i18next';
import { Card, Typography } from 'antd';
import ErrorInPlace from 'components/ErrorInPlace/ErrorInPlace';
import Spinner from 'components/Spinner';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { WidgetGraphContainerProps } from 'container/NewWidget/types';
import APIError from 'types/api/error';
import { getSortedSeriesData } from 'utils/getSortedSeriesData';

import { NotFoundContainer } from './styles';
import { populateMultipleResults } from './util';
import WidgetGraph from './WidgetGraphs';

function WidgetGraphContainer({
	selectedGraph,
	queryResponse,
	setRequestData,
	selectedWidget,
	isLoadingPanelData,
	enableDrillDown = false,
}: WidgetGraphContainerProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	if (queryResponse.data && selectedGraph === PANEL_TYPES.BAR) {
		const sortedSeriesData = getSortedSeriesData(
			queryResponse.data?.payload.data.result,
		);
		queryResponse.data.payload.data.result = sortedSeriesData;
	}

	if (queryResponse.data && selectedGraph === PANEL_TYPES.PIE) {
		const transformedData = populateMultipleResults(queryResponse?.data);
		queryResponse.data = transformedData;
	}

	if (selectedWidget === undefined) {
		return <Card>{t('invalid_widget')}</Card>;
	}

	if (queryResponse?.error) {
		return (
			<NotFoundContainer>
				<ErrorInPlace error={queryResponse.error as APIError} />
			</NotFoundContainer>
		);
	}
	if (queryResponse.isLoading && selectedGraph !== PANEL_TYPES.LIST) {
		return <Spinner size="large" tip="Loading..." />;
	}

	if (isLoadingPanelData) {
		return <Spinner size="large" tip="Loading..." />;
	}

	if (queryResponse.isIdle) {
		return (
			<NotFoundContainer>
				<Typography>{t('no_data_widget')}</Typography>
			</NotFoundContainer>
		);
	}

	return (
		<WidgetGraph
			selectedWidget={selectedWidget}
			queryResponse={queryResponse}
			setRequestData={setRequestData}
			selectedGraph={selectedGraph}
			enableDrillDown={enableDrillDown}
		/>
	);
}

export default WidgetGraphContainer;
