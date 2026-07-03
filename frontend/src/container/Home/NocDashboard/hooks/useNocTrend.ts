import { TrendMetric, TrendSeries, TrendTarget } from '../types';

export interface UseNocTrendResult {
	series: TrendSeries[];
	stepSec: number;
	isLoading: boolean;
	isError: boolean;
}

// SEED STUB — 본문은 Lane A가 채움(impl-plan Task 2). 시그니처·반환 타입은 계약(불변).
export default function useNocTrend(
	_targets: TrendTarget[],
	_metric: TrendMetric,
): UseNocTrendResult {
	return { series: [], stepSec: 60, isLoading: false, isError: false };
}
