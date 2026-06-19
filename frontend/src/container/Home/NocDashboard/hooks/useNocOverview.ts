import { ENTITY_VERSION_V4 } from 'constants/app';
import { FeatureKeys } from 'constants/features';
import {
	getQueryRangeRequestData,
	getServiceListFromQuery,
} from 'container/ServiceApplication/utils';
import { useGetQueriesRange } from 'hooks/queryBuilder/useGetQueriesRange';
import useGetTopLevelOperations from 'hooks/useGetTopLevelOperations';
import useResourceAttribute from 'hooks/useResourceAttribute';
import { convertRawQueriesToTraceSelectedTags } from 'hooks/useResourceAttribute/utils';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { QueryKey } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { useAppContext } from 'providers/App/App';
import { AppState } from 'store/reducers';
import { ServicesList } from 'types/api/metrics/getService';
import { GlobalReducer } from 'types/reducer/globalTime';
import { Tags } from 'types/reducer/trace';

import { NocGoldenSignal, NocHealth, NocKpi, NocServiceRow } from '../types';

const HOME_INTERVAL = 30 * 60 * 1000;

// error rate thresholds (percent)
const HEALTHY_ERR_PCT = 1;
const CRITICAL_ERR_PCT = 5;

// max rows shown in the service health list
const MAX_SERVICE_ROWS = 12;

function healthFromErr(errPct: number): NocHealth {
	if (errPct >= CRITICAL_ERR_PCT) {
		return 'critical';
	}
	if (errPct >= HEALTHY_ERR_PCT) {
		return 'warning';
	}
	return 'healthy';
}

// p99 arrives in nanoseconds (the Services table renders it as value / 1e6).
function p99ToMs(p99: number): number {
	if (!Number.isFinite(p99) || p99 <= 0) {
		return 0;
	}
	return p99 / 1e6;
}

function formatRps(rps: number): string {
	if (rps >= 1000) {
		return `${(rps / 1000).toFixed(1)}k`;
	}
	return rps >= 10 ? `${Math.round(rps)}` : rps.toFixed(1);
}

interface Aggregate {
	totalRps: number;
	weightedErrPct: number;
	maxP99Ms: number;
	healthy: number;
	total: number;
}

function aggregate(services: ServicesList[]): Aggregate {
	let totalRps = 0;
	let errNumerator = 0;
	let maxP99Ms = 0;
	let healthy = 0;

	services.forEach((s) => {
		const rps = Number.isFinite(s.callRate) ? s.callRate : 0;
		const err = Number.isFinite(s.errorRate) ? s.errorRate : 0;
		totalRps += rps;
		errNumerator += err * rps;
		maxP99Ms = Math.max(maxP99Ms, p99ToMs(s.p99));
		if (err < HEALTHY_ERR_PCT) {
			healthy += 1;
		}
	});

	return {
		totalRps,
		weightedErrPct: totalRps > 0 ? errNumerator / totalRps : 0,
		maxP99Ms,
		healthy,
		total: services.length,
	};
}

export interface UseNocOverviewResult {
	kpis: NocKpi[];
	golden: NocGoldenSignal[];
	services: NocServiceRow[];
	isLoading: boolean;
	isError: boolean;
}

export default function useNocOverview(firingCount: number): UseNocOverviewResult {
	const { t } = useTranslation('home');
	const { selectedTime: globalSelectedInterval } = useSelector<
		AppState,
		GlobalReducer
	>((state) => state.globalTime);

	const { featureFlags } = useAppContext();
	const dotMetricsEnabled =
		featureFlags?.find((flag) => flag.name === FeatureKeys.DOT_METRICS_ENABLED)
			?.active || false;

	const { queries } = useResourceAttribute();
	const selectedTags = useMemo(
		() => (convertRawQueriesToTraceSelectedTags(queries) as Tags[]) || [],
		[queries],
	);

	const timeRange = useMemo(() => {
		const now = Date.now();
		return { startTime: now - HOME_INTERVAL, endTime: now };
	}, []);

	const topLevelQueryKey: QueryKey = useMemo(
		() => [
			timeRange.startTime,
			timeRange.endTime,
			selectedTags,
			globalSelectedInterval,
		],
		[timeRange.startTime, timeRange.endTime, selectedTags, globalSelectedInterval],
	);

	const {
		data: topLevelData,
		isLoading: isTopLevelLoading,
		isError: isTopLevelError,
	} = useGetTopLevelOperations(topLevelQueryKey, {
		start: timeRange.startTime * 1e6,
		end: timeRange.endTime * 1e6,
	});

	const topLevelOperations = useMemo(
		() => Object.entries(topLevelData || {}),
		[topLevelData],
	);

	const queryRangeRequestData = useMemo(
		() =>
			getQueryRangeRequestData({
				topLevelOperations,
				globalSelectedInterval,
				dotMetricsEnabled,
			}),
		[topLevelOperations, globalSelectedInterval, dotMetricsEnabled],
	);

	const dataQueries = useGetQueriesRange(queryRangeRequestData, ENTITY_VERSION_V4, {
		queryKey: useMemo(
			() => [
				`noc-overview-${globalSelectedInterval}`,
				timeRange.endTime,
				timeRange.startTime,
				globalSelectedInterval,
			],
			[globalSelectedInterval, timeRange.endTime, timeRange.startTime],
		),
		keepPreviousData: true,
		enabled: topLevelOperations.length > 0,
		refetchOnMount: false,
	});

	const isQueriesLoading = useMemo(
		() => dataQueries.some((query) => query.isLoading),
		[dataQueries],
	);
	const isQueriesError = useMemo(
		() => dataQueries.some((query) => query.isError),
		[dataQueries],
	);

	const services: ServicesList[] = useMemo(
		() =>
			getServiceListFromQuery({
				queries: dataQueries,
				topLevelOperations,
				isLoading: isQueriesLoading,
			}),
		[dataQueries, topLevelOperations, isQueriesLoading],
	);

	const agg = useMemo(() => aggregate(services), [services]);

	const serviceRows = useMemo<NocServiceRow[]>(
		() =>
			[...services]
				.map((s) => ({
					name: s.serviceName,
					health: healthFromErr(Number.isFinite(s.errorRate) ? s.errorRate : 0),
					p99Ms: p99ToMs(s.p99),
					errPct: Number.isFinite(s.errorRate) ? s.errorRate : 0,
					rps: Number.isFinite(s.callRate) ? s.callRate : 0,
				}))
				.sort((a, b) => b.rps - a.rps)
				.slice(0, MAX_SERVICE_ROWS),
		[services],
	);

	const isLoading = isTopLevelLoading || isQueriesLoading;
	const isError = isTopLevelError || isQueriesError;

	const kpis = useMemo<NocKpi[]>(() => {
		const uptimePct = agg.total > 0 ? (agg.healthy / agg.total) * 100 : 100;
		return [
			{
				key: 'uptime',
				label: '가동률',
				value: uptimePct.toFixed(1),
				unit: '%',
				accent: uptimePct >= 99 ? 'ok' : 'neutral',
			},
			{
				key: 'alerts',
				label: '활성 알림',
				value: String(firingCount),
				delta:
					firingCount > 0
						? t('noc_kpi_alerts_firing', { count: firingCount })
						: t('noc_kpi_alerts_none'),
				deltaDir: firingCount > 0 ? 'up' : 'flat',
				accent: firingCount > 0 ? 'brand' : 'neutral',
			},
			{
				key: 'errorRate',
				label: '에러율',
				value: agg.weightedErrPct.toFixed(2),
				unit: '%',
				accent: agg.weightedErrPct >= HEALTHY_ERR_PCT ? 'error' : 'ok',
			},
			{
				key: 'p99',
				label: 'P99',
				value: String(Math.round(agg.maxP99Ms)),
				unit: 'ms',
				accent: 'neutral',
			},
			{
				key: 'rps',
				label: 'RPS',
				value: formatRps(agg.totalRps),
				accent: 'neutral',
			},
			{
				key: 'services',
				label: '서비스',
				value: `${agg.healthy}/${agg.total}`,
				accent: agg.healthy < agg.total ? 'error' : 'ok',
			},
		];
	}, [agg, firingCount, t]);

	const golden = useMemo<NocGoldenSignal[]>(
		() => [
			{ key: 'latency', label: '지연', value: `${Math.round(agg.maxP99Ms)}ms` },
			{ key: 'traffic', label: '트래픽', value: formatRps(agg.totalRps) },
			{
				key: 'errors',
				label: '에러',
				value: `${agg.weightedErrPct.toFixed(2)}%`,
				accent: agg.weightedErrPct >= HEALTHY_ERR_PCT ? 'error' : undefined,
			},
			{ key: 'firing', label: '발화', value: String(firingCount) },
		],
		[agg, firingCount],
	);

	return { kpis, golden, services: serviceRows, isLoading, isError };
}
