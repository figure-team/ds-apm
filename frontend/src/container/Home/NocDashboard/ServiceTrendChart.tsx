import { useIsDarkMode } from 'hooks/useDarkMode';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import TrendLegend, { TrendLegendItem } from './components/trend/TrendLegend';
import TrendPlot from './components/trend/TrendPlot';
import TrendToolbar from './components/trend/TrendToolbar';
import useHiddenServices from './hooks/useHiddenServices';
import { ResolvedTrendSeries, TrendMetric, TrendSeries } from './types';
import { SERIES_PALETTE_LIGHT } from './utils/deriveState';

export interface ServiceTrendChartProps {
	series: TrendSeries[];
	metric: TrendMetric;
	onMetricChange: (m: TrendMetric) => void;
	thresholdLine?: number;
	loading: boolean;
	error: boolean;
}

// 상태(hover·숨김·로그축)와 색 확정만 담당하는 컨테이너 — 렌더는 trend/ 리프 컴포넌트로 위임.
export default function ServiceTrendChart({
	series,
	metric,
	onMetricChange,
	thresholdLine,
	loading,
	error,
}: ServiceTrendChartProps): JSX.Element {
	const { t } = useTranslation('home');
	const isDark = useIsDarkMode();
	const [hovered, setHovered] = useState<string | null>(null);
	const [logScale, setLogScale] = useState(false);
	const { hidden, toggle } = useHiddenServices();

	const resolved: ResolvedTrendSeries[] = useMemo(
		() =>
			series.map((s, i) => ({
				...s,
				resolvedColor: isDark
					? s.color
					: SERIES_PALETTE_LIGHT[i % SERIES_PALETTE_LIGHT.length],
			})),
		[series, isDark],
	);
	// 숨긴 계열은 스케일 계산에서도 빠져야 이상치(load-generator 등) 제외 효과가 난다
	const visible = useMemo(() => resolved.filter((s) => !hidden.has(s.name)), [
		resolved,
		hidden,
	]);
	const legendItems: TrendLegendItem[] = resolved.map((s) => ({
		name: s.name,
		color: s.missing ? s.color : s.resolvedColor,
		missing: s.missing,
		hidden: hidden.has(s.name),
	}));

	const renderBody = (): JSX.Element => {
		if (loading) {
			return (
				<div className="noc-c2-trend-body">
					<div className="noc-c2-trend-msg">{t('noc_c2_trend_loading')}</div>
				</div>
			);
		}
		if (error) {
			return (
				<div className="noc-c2-trend-body">
					<div className="noc-c2-trend-msg">{t('noc_c2_trend_error')}</div>
				</div>
			);
		}
		return (
			<TrendPlot
				series={visible}
				metric={metric}
				logScale={logScale}
				thresholdLine={thresholdLine}
				hovered={hovered}
			/>
		);
	};

	return (
		<div className="noc-c2-trend">
			<div className="noc-c2-trend-head">
				<TrendLegend
					items={legendItems}
					hovered={hovered}
					onHover={setHovered}
					onToggle={toggle}
				/>
				<TrendToolbar
					metric={metric}
					onMetricChange={onMetricChange}
					logScale={logScale}
					onLogScaleChange={setLogScale}
				/>
			</div>
			{renderBody()}
		</div>
	);
}
