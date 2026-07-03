import { useIsDarkMode } from 'hooks/useDarkMode';
import { useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { TrendMetric, TrendSeries } from '../types';
import { SERIES_PALETTE_LIGHT } from '../utils/deriveState';

const VB_W = 1000;
const VB_H = 320;
const PAD = { top: 16, right: 96, bottom: 24, left: 44 }; // right 넓게: 직접 라벨 공간

export interface Scale {
	minT: number;
	maxT: number;
	minV: number;
	maxV: number;
}

export function computeScale(series: TrendSeries[], metric: TrendMetric): Scale {
	const pts = series.filter((s) => !s.missing).flatMap((s) => s.points);
	if (pts.length === 0) {
		return { minT: 0, maxT: 1, minV: 0, maxV: metric === 'err' ? 2 : 1 };
	}
	const ts = pts.map((p) => p.t);
	const vs = pts.map((p) => p.v);
	const minV = Math.min(0, ...vs);
	let maxV = Math.max(...vs);
	if (maxV <= minV) {
		maxV = minV + 1;
	}
	maxV *= 1.1; // headroom
	return { minT: Math.min(...ts), maxT: Math.max(...ts), minV, maxV };
}

const METRICS: TrendMetric[] = ['err', 'p99', 'rps'];

export interface ServiceTrendChartProps {
	series: TrendSeries[];
	metric: TrendMetric;
	onMetricChange: (m: TrendMetric) => void;
	thresholdLine?: number;
	loading: boolean;
	error: boolean;
}

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
	const wrapRef = useRef<HTMLDivElement>(null);

	const scale = useMemo(() => computeScale(series, metric), [series, metric]);

	const x = (tms: number): number =>
		PAD.left +
		((tms - scale.minT) / (scale.maxT - scale.minT || 1)) *
			(VB_W - PAD.left - PAD.right);
	const y = (v: number): number =>
		VB_H -
		PAD.bottom -
		((v - scale.minV) / (scale.maxV - scale.minV || 1)) *
			(VB_H - PAD.top - PAD.bottom);

	const colorOf = (s: TrendSeries, i: number): string =>
		isDark ? s.color : SERIES_PALETTE_LIGHT[i % SERIES_PALETTE_LIGHT.length];

	const toolbar = (
		<div className="noc-c2-trend-tabs">
			{METRICS.map((m) => (
				<button
					key={m}
					type="button"
					className={m === metric ? 'active' : ''}
					onClick={(): void => onMetricChange(m)}
				>
					{t(`noc_c2_metric_${m}`)}
				</button>
			))}
		</div>
	);

	const legend = (
		<div className="noc-c2-trend-legend">
			{series.map((s, i) => (
				<button
					key={s.name}
					type="button"
					className={`noc-c2-legend-item${s.missing ? ' missing' : ''}${
						hovered && hovered !== s.name ? ' dim' : ''
					}`}
					onMouseEnter={(): void => setHovered(s.name)}
					onMouseLeave={(): void => setHovered(null)}
				>
					<span
						className="noc-c2-legend-swatch"
						style={{ background: s.missing ? 'var(--noc-c-sec)' : colorOf(s, i) }}
					/>
					<span className="noc-c2-legend-name">{s.name}</span>
					{s.missing ? (
						<span className="noc-c2-legend-nodata">{t('noc_c2_series_nodata')}</span>
					) : null}
				</button>
			))}
		</div>
	);

	const renderBody = (): JSX.Element => {
		if (loading) {
			return <div className="noc-c2-trend-msg">{t('noc_c2_trend_loading')}</div>;
		}
		if (error) {
			return <div className="noc-c2-trend-msg">{t('noc_c2_trend_error')}</div>;
		}
		return (
			<svg
				className="noc-c2-trend-svg"
				viewBox={`0 0 ${VB_W} ${VB_H}`}
				preserveAspectRatio="none"
			>
				{metric === 'err' && thresholdLine !== undefined ? (
					<line
						x1={PAD.left}
						x2={VB_W - PAD.right}
						y1={y(thresholdLine)}
						y2={y(thresholdLine)}
						className="noc-c2-threshold"
						strokeDasharray="4 4"
					/>
				) : null}
				{series.map((s, i) => {
					if (s.missing || s.points.length === 0) {
						return null;
					}
					const d = s.points
						.map(
							(p, j) =>
								`${j === 0 ? 'M' : 'L'}${x(p.t).toFixed(1)},${y(p.v).toFixed(1)}`,
						)
						.join(' ');
					const last = s.points[s.points.length - 1];
					const dim = hovered && hovered !== s.name;
					return (
						<g key={s.name} opacity={dim ? 0.18 : 1}>
							<path d={d} fill="none" stroke={colorOf(s, i)} strokeWidth={2} />
							<circle cx={x(last.t)} cy={y(last.v)} r={3} fill={colorOf(s, i)} />
							<text
								x={x(last.t) + 6}
								y={y(last.v)}
								className="noc-c2-endlabel"
								fill={colorOf(s, i)}
							>
								{s.name} {last.v.toFixed(metric === 'err' ? 1 : 0)}
							</text>
						</g>
					);
				})}
			</svg>
		);
	};

	return (
		<div className="noc-c2-trend" ref={wrapRef}>
			<div className="noc-c2-trend-head">
				{legend}
				{toolbar}
			</div>
			{renderBody()}
		</div>
	);
}
