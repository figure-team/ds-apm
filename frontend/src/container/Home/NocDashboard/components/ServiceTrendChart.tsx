import { TrendMetric, TrendSeries } from '../types';

export interface Scale {
	minT: number;
	maxT: number;
	minV: number;
	maxV: number;
}

// SEED STUB — 본문은 Lane B가 채움(impl-plan Task 7). named export도 계약(불변).
export function computeScale(_series: TrendSeries[], _metric: TrendMetric): Scale {
	return { minT: 0, maxT: 1, minV: 0, maxV: 1 };
}

export interface ServiceTrendChartProps {
	series: TrendSeries[];
	metric: TrendMetric;
	onMetricChange: (m: TrendMetric) => void;
	thresholdLine?: number;
	loading: boolean;
	error: boolean;
}

// SEED STUB — 본문은 Lane B가 채움(impl-plan Task 7). props 인터페이스는 계약(불변).
export default function ServiceTrendChart(
	_props: ServiceTrendChartProps,
): JSX.Element {
	return <div className="noc-c2-trend" data-stub="trend" />;
}
