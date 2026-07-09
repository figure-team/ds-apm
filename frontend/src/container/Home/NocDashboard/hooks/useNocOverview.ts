import getService from 'api/metrics/getService';
import useResourceAttribute from 'hooks/useResourceAttribute';
import { convertRawQueriesToTraceSelectedTags } from 'hooks/useResourceAttribute/utils';
import GetMinMax from 'lib/getMinMax';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { AppState } from 'store/reducers';
import { ServicesList } from 'types/api/metrics/getService';
import { GlobalReducer } from 'types/reducer/globalTime';
import { Tags } from 'types/reducer/trace';

import { NocHealth, NocKpi, NocServiceRow } from '../types';

// error rate thresholds (percent)
const HEALTHY_ERR_PCT = 1;
const CRITICAL_ERR_PCT = 5;

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
	services: NocServiceRow[];
	isLoading: boolean;
	isError: boolean;
}

export default function useNocOverview(firingCount: number): UseNocOverviewResult {
	const { t } = useTranslation('home');
	const {
		maxTime,
		minTime,
		selectedTime: globalSelectedInterval,
	} = useSelector<AppState, GlobalReducer>((state) => state.globalTime);

	const { queries } = useResourceAttribute();
	const selectedTags = useMemo(
		() => (convertRawQueriesToTraceSelectedTags(queries) as Tags[]) || [],
		[queries],
	);

	// Reuse the same /api/v2/services endpoint that powers the Services list page.
	// The previous query_range path rendered all-zero because TABLE-panel responses
	// are reshaped (no `series`), which getSeriesValue could not read.
	const {
		data: servicesData,
		isLoading,
		isError,
	} = useQuery<ServicesList[] | undefined>(
		['noc-overview-services', minTime, maxTime, globalSelectedInterval, selectedTags],
		() => {
			const { minTime: start, maxTime: end } = GetMinMax(globalSelectedInterval, [
				minTime / 1e6,
				maxTime / 1e6,
			]);
			return getService({ start, end, selectedTags });
		},
		{
			keepPreviousData: true,
		},
	);

	const services: ServicesList[] = useMemo(() => servicesData || [], [
		servicesData,
	]);

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
				.sort((a, b) => b.rps - a.rps),
		[services],
	);

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

	return { kpis, services: serviceRows, isLoading, isError };
}
