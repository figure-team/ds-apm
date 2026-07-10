import { ENTITY_VERSION_V5 } from 'constants/app';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { useGetQueryRange } from 'hooks/queryBuilder/useGetQueryRange';
import { useMemo } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { AppState } from 'store/reducers';
import { DataSource } from 'types/common/queryBuilder';
import { GlobalReducer } from 'types/reducer/globalTime';

import { TrendMetric, TrendPoint, TrendSeries, TrendTarget } from '../types';

const STEP_MIN_SEC = 30;
const STEP_MAX_SEC = 300;
const POINTS_TARGET = 45; // §5.2 stepSec = clamp(round(windowSec/45), MIN, MAX)

function deriveStep(windowSec: number): number {
	const raw = Math.round(windowSec / POINTS_TARGET);
	return Math.min(STEP_MAX_SEC, Math.max(STEP_MIN_SEC, raw || STEP_MIN_SEC));
}

// service.name IN ('a','b') 필터 표현식
function inFilter(names: string[]): string {
	const quoted = names.map((n) => `'${n.replace(/'/g, "\\'")}'`).join(', ');
	return `service.name IN [${quoted}]`;
}

// Step 1 실측: 앱 미구동 상태라 라이브 검증 불가. useGetQueryRange → GetMetricQueryRange가
// 정규화한 v3 shape(`data.payload.data.result[]`)를 가정한다. 각 계열 라벨은
// result[i].metric['service.name'](fallback labels), 값은 values[[tsSec, "v"]].
// 라이브 실측 시 이 경로가 다르면 NormPayloadData/svcName만 조정(파서 로직 불변).
interface NormResult {
	queryName: string;
	metric?: Record<string, string>;
	labels?: Record<string, string>;
	values: [number, string][];
}
interface NormPayloadData {
	result: NormResult[];
}

function svcName(r: NormResult): string {
	return r.metric?.['service.name'] ?? r.labels?.['service.name'] ?? '';
}

// timestamp(sec) -> value 맵으로 A/B/C 조인
function toMap(r: NormResult | undefined): Map<number, number> {
	const m = new Map<number, number>();
	r?.values.forEach(([ts, v]) => {
		const num = Number(v);
		m.set(ts, Number.isFinite(num) ? num : 0);
	});
	return m;
}

// 백엔드 v5 응답의 values 순서는 시간순이 보장되지 않는다(실측: 홈 트렌드 실타래 렌더).
// 선으로 잇기 전에 시간 오름차순 정렬하고, 동일 타임스탬프는 마지막 값을 채택한다.
export function normalizePoints(points: TrendPoint[]): TrendPoint[] {
	const byT = new Map<number, number>();
	points.forEach((p) => byT.set(p.t, p.v));
	return [...byT.entries()]
		.sort((a, b) => a[0] - b[0])
		.map(([t, v]) => ({ t, v }));
}

export function parseTrendSeries(
	data: NormPayloadData | undefined,
	targets: TrendTarget[],
	metric: TrendMetric,
	stepSec: number,
): TrendSeries[] {
	const results = data?.result ?? [];
	const byQuery = (q: string): Map<string, NormResult> => {
		const m = new Map<string, NormResult>();
		results
			.filter((r) => r.queryName === q)
			.forEach((r) => m.set(svcName(r), r));
		return m;
	};
	const A = byQuery('A'); // count -> rps
	const B = byQuery('B'); // count has_error -> err numerator
	const C = byQuery('C'); // p99 ns

	return targets.map((tg) => {
		const a = A.get(tg.name);
		const b = B.get(tg.name);
		const c = C.get(tg.name);

		let source: NormResult | undefined;
		let points: TrendPoint[] = [];

		if (metric === 'rps') {
			source = a;
			points =
				a?.values.map(([ts, v]) => ({ t: ts * 1000, v: Number(v) / stepSec })) ?? [];
		} else if (metric === 'p99') {
			source = c;
			points =
				c?.values.map(([ts, v]) => ({ t: ts * 1000, v: Number(v) / 1e6 })) ?? [];
		} else {
			// err = B/A*100, B 결측 서비스는 0으로 조인(§5.2)
			source = a;
			const bMap = toMap(b);
			points =
				a?.values.map(([ts, v]) => {
					const denom = Number(v);
					const numer = bMap.get(ts) ?? 0;
					return { t: ts * 1000, v: denom > 0 ? (numer / denom) * 100 : 0 };
				}) ?? [];
		}

		return {
			name: tg.name,
			color: tg.color,
			points: normalizePoints(points),
			missing: !source, // A(rps/err) 또는 C(p99) series 자체가 없으면 missing
		};
	});
}

function buildQueryData(names: string[]): unknown[] {
	const groupBy = [{ key: 'service.name', dataType: 'string', type: 'resource' }];
	const base = {
		dataSource: DataSource.TRACES,
		timeAggregation: 'rate',
		spaceAggregation: 'sum',
		functions: [],
		disabled: false, // 3쿼리 모두 series 직접 읽음 (참조 utils는 formula용이라 C를 disabled)
		having: [],
		limit: null,
		orderBy: [],
		groupBy,
		legend: '',
		reduceTo: 'avg',
	};
	return [
		{
			...base,
			queryName: 'A',
			aggregateOperator: 'count',
			aggregations: [{ expression: 'count()' }],
			filter: { expression: inFilter(names) },
			expression: 'A',
		},
		{
			...base,
			queryName: 'B',
			aggregateOperator: 'count',
			aggregations: [{ expression: 'count()' }],
			filter: { expression: `${inFilter(names)} AND has_error = true` },
			expression: 'B',
		},
		{
			...base,
			queryName: 'C',
			aggregateOperator: 'p99',
			timeAggregation: 'p99',
			aggregations: [{ expression: 'p99(duration_nano)' }],
			filter: { expression: inFilter(names) },
			expression: 'C',
		},
	];
}

export interface UseNocTrendResult {
	series: TrendSeries[];
	stepSec: number;
	isLoading: boolean;
	isError: boolean;
}

export default function useNocTrend(
	targets: TrendTarget[],
	metric: TrendMetric,
): UseNocTrendResult {
	const { maxTime, minTime, selectedTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);

	const names = useMemo(() => targets.map((t) => t.name), [targets]);
	const windowSec = Math.max(1, Math.round((maxTime - minTime) / 1e9)); // ns -> s
	const stepSec = deriveStep(windowSec);

	const requestData = useMemo(
		() => ({
			selectedTime,
			graphType: PANEL_TYPES.TIME_SERIES, // TABLE 아님 — TABLE엔 series 없음
			query: {
				builder: {
					queryData: buildQueryData(names),
					queryFormulas: [],
				},
				clickhouse_sql: [],
				id: 'noc-trend',
				promql: [],
				queryType: 'builder',
			},
			params: { dataSource: DataSource.TRACES },
			start: Math.floor(minTime / 1e6),
			end: Math.floor(maxTime / 1e6),
			step: stepSec,
		}),
		[names, selectedTime, minTime, maxTime, stepSec],
	);

	const { data, isLoading, isError } = useGetQueryRange(
		requestData as never,
		ENTITY_VERSION_V5,
		{
			enabled: names.length > 0,
			queryKey: ['noc-trend', metric, stepSec, names, minTime, maxTime],
			keepPreviousData: true,
		},
	);

	const series = useMemo(
		() =>
			parseTrendSeries(
				(data?.payload?.data as unknown) as NormPayloadData | undefined,
				targets,
				metric,
				stepSec,
			),
		[data, targets, metric, stepSec],
	);

	return { series, stepSec, isLoading, isError };
}
