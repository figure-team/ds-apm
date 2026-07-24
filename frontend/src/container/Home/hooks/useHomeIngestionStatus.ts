import { useEffect, useState } from 'react';
import logEvent from 'api/common/logEvent';
import { useGetMetricsOnboardingStatus } from 'api/generated/services/metrics';
import { ENTITY_VERSION_V5 } from 'constants/app';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { DEFAULT_TIME_RANGE } from 'container/TopNav/DateTimeSelectionV2/constants';
import { useGetQueryRange } from 'hooks/queryBuilder/useGetQueryRange';
import { DataSource } from 'types/common/queryBuilder';
import { isIngestionActive } from 'utils/app';

const homeInterval = 30 * 60 * 1000;

// Toggle: render the NOC/control-center dashboard once telemetry is flowing.
// New workspaces (no data yet) keep the onboarding checklist experience.
const USE_NOC_HOME = true;

export interface HomeIngestionStatus {
	isLogsIngestionActive: boolean;
	isTracesIngestionActive: boolean;
	isMetricsIngestionActive: boolean;
	isAnyIngestionActive: boolean;
	showNocDashboard: boolean;
	isLogsLoading: boolean;
	isTracesLoading: boolean;
}

/**
 * Detects which telemetry signals have started flowing so the home page can
 * choose between the NOC dashboard and the onboarding checklist. Each flag
 * latches on: once a signal is seen it stays active for the session.
 */
export default function useHomeIngestionStatus(): HomeIngestionStatus {
	const [startTime, setStartTime] = useState<number | null>(null);
	const [endTime, setEndTime] = useState<number | null>(null);

	useEffect(() => {
		const now = new Date();
		const startTime = new Date(now.getTime() - homeInterval);
		const endTime = now;

		setStartTime(startTime.getTime());
		setEndTime(endTime.getTime());
	}, []);

	// Detect Logs
	const { data: logsData, isLoading: isLogsLoading } = useGetQueryRange(
		{
			query: initialQueriesMap[DataSource.LOGS],
			graphType: PANEL_TYPES.VALUE,
			selectedTime: 'GLOBAL_TIME',
			globalSelectedInterval: DEFAULT_TIME_RANGE,
			params: {
				dataSource: DataSource.LOGS,
			},
			formatForWeb: false,
		},
		ENTITY_VERSION_V5,
		{
			queryKey: [
				REACT_QUERY_KEY.GET_QUERY_RANGE,
				DEFAULT_TIME_RANGE,
				endTime || Date.now(),
				startTime || Date.now(),
				initialQueriesMap[DataSource.LOGS],
			],
			enabled: !!startTime && !!endTime,
		},
	);

	// Detect Traces
	const { data: tracesData, isLoading: isTracesLoading } = useGetQueryRange(
		{
			query: initialQueriesMap[DataSource.TRACES],
			graphType: PANEL_TYPES.VALUE,
			selectedTime: 'GLOBAL_TIME',
			globalSelectedInterval: DEFAULT_TIME_RANGE,
			params: {
				dataSource: DataSource.TRACES,
			},
			formatForWeb: false,
		},
		ENTITY_VERSION_V5,
		{
			queryKey: [
				REACT_QUERY_KEY.GET_QUERY_RANGE,
				DEFAULT_TIME_RANGE,
				endTime || Date.now(),
				startTime || Date.now(),
				initialQueriesMap[DataSource.TRACES],
			],
			enabled: !!startTime && !!endTime,
		},
	);

	// Detect Metrics
	const { data: metricsOnboardingData } = useGetMetricsOnboardingStatus();

	const [isLogsIngestionActive, setIsLogsIngestionActive] = useState(false);
	const [isTracesIngestionActive, setIsTracesIngestionActive] = useState(false);
	const [isMetricsIngestionActive, setIsMetricsIngestionActive] =
		useState(false);

	useEffect(() => {
		if (isIngestionActive(logsData?.payload)) {
			setIsLogsIngestionActive(true);
		}
	}, [logsData]);

	useEffect(() => {
		if (isIngestionActive(tracesData?.payload)) {
			setIsTracesIngestionActive(true);
		}
	}, [tracesData]);

	useEffect(() => {
		if (metricsOnboardingData?.data?.hasMetrics) {
			setIsMetricsIngestionActive(true);
		}
	}, [metricsOnboardingData]);

	useEffect(() => {
		logEvent('Homepage: Visited', {});
	}, []);

	const isAnyIngestionActive =
		isLogsIngestionActive || isTracesIngestionActive || isMetricsIngestionActive;

	return {
		isLogsIngestionActive,
		isTracesIngestionActive,
		isMetricsIngestionActive,
		isAnyIngestionActive,
		showNocDashboard: USE_NOC_HOME && isAnyIngestionActive,
		isLogsLoading,
		isTracesLoading,
	};
}
