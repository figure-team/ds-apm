import { useIsDarkMode } from 'hooks/useDarkMode';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { TrendMetric, TrendSeries } from '../types';
import { SERIES_PALETTE_LIGHT } from '../utils/deriveState';

// 우측 여백은 끝점 직접 라벨(서비스명+값) 공간
const PAD = { top: 14, right: 170, bottom: 10, left: 52 };
const LABEL_GAP = 15; // 끝점 라벨 최소 세로 간격(px)
const NAME_MAX = 18; // 라벨 서비스명 최대 길이
const TICKS = 4; // Y축 눈금 수(구간)

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

const METRIC_UNIT: Record<TrendMetric, string> = { err: '%', p99: 'ms', rps: '' };

function truncName(name: string): string {
	return name.length > NAME_MAX ? `${name.slice(0, NAME_MAX - 1)}…` : name;
}

function formatValue(v: number, metric: TrendMetric): string {
	return metric === 'err' ? v.toFixed(1) : String(Math.round(v));
}

// 끝점 직접 라벨 세로 스택 — 겹치면 아래로 밀고, 바닥을 넘치면 위로 되민다(§4.3 겹침 자동 회피).
function stackLabelYs(ideal: number[], minY: number, maxY: number): number[] {
	const order = ideal
		.map((y, i) => ({ y: Math.min(Math.max(y, minY), maxY), i }))
		.sort((a, b) => a.y - b.y);
	const placed: number[] = [];
	order.forEach((e, k) => {
		placed.push(k === 0 ? e.y : Math.max(e.y, placed[k - 1] + LABEL_GAP));
	});
	for (let k = order.length - 1; k >= 0; k -= 1) {
		const cap = k === order.length - 1 ? maxY : placed[k + 1] - LABEL_GAP;
		if (placed[k] > cap) {
			placed[k] = cap;
		}
	}
	const out: number[] = [];
	order.forEach((e, k) => {
		out[e.i] = placed[k];
	});
	return out;
}

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
	const bodyRef = useRef<HTMLDivElement>(null);
	// viewBox 스트레치는 텍스트를 왜곡시키므로 컨테이너 실측 픽셀로 1:1 렌더.
	const [size, setSize] = useState({ w: 860, h: 300 });

	useEffect(() => {
		const el = bodyRef.current;
		if (!el || typeof ResizeObserver === 'undefined') {
			return undefined;
		}
		const ro = new ResizeObserver((entries) => {
			const r = entries[0]?.contentRect;
			if (r && r.width > 0 && r.height > 0) {
				setSize({ w: r.width, h: r.height });
			}
		});
		ro.observe(el);
		return (): void => ro.disconnect();
	}, []);

	const scale = useMemo(() => computeScale(series, metric), [series, metric]);
	const { w, h } = size;

	const x = (tms: number): number =>
		PAD.left +
		((tms - scale.minT) / (scale.maxT - scale.minT || 1)) *
			(w - PAD.left - PAD.right);
	const y = (v: number): number =>
		h -
		PAD.bottom -
		((v - scale.minV) / (scale.maxV - scale.minV || 1)) *
			(h - PAD.top - PAD.bottom);

	const colorOf = (s: TrendSeries, i: number): string =>
		isDark ? s.color : SERIES_PALETTE_LIGHT[i % SERIES_PALETTE_LIGHT.length];

	const drawable = series
		.map((s, i) => ({ s, i }))
		.filter(({ s }) => !s.missing && s.points.length > 0);
	const labelYs = stackLabelYs(
		drawable.map(({ s }) => y(s.points[s.points.length - 1].v) + 4),
		PAD.top + 6,
		h - PAD.bottom - 2,
	);

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
			<svg className="noc-c2-trend-svg" width={w} height={h}>
				{/* Y축 눈금 + 그리드 */}
				{Array.from({ length: TICKS + 1 }, (_, k) => {
					const v = scale.minV + ((scale.maxV - scale.minV) * k) / TICKS;
					const ty = y(v);
					return (
						<g key={`tick-${v}`} className="noc-c2-tick">
							<line x1={PAD.left} x2={w - PAD.right} y1={ty} y2={ty} />
							<text x={PAD.left - 8} y={ty + 3} textAnchor="end">
								{formatValue(v, metric)}
								{METRIC_UNIT[metric]}
							</text>
						</g>
					);
				})}
				{metric === 'err' && thresholdLine !== undefined ? (
					<line
						x1={PAD.left}
						x2={w - PAD.right}
						y1={y(thresholdLine)}
						y2={y(thresholdLine)}
						className="noc-c2-threshold"
						strokeDasharray="4 4"
					/>
				) : null}
				{drawable.map(({ s, i }, k) => {
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
								x={x(last.t) + 8}
								y={labelYs[k]}
								className="noc-c2-endlabel"
								fill={colorOf(s, i)}
							>
								{truncName(s.name)} {formatValue(last.v, metric)}
							</text>
						</g>
					);
				})}
			</svg>
		);
	};

	return (
		<div className="noc-c2-trend">
			<div className="noc-c2-trend-head">
				{legend}
				{toolbar}
			</div>
			<div className="noc-c2-trend-body" ref={bodyRef}>
				{renderBody()}
			</div>
		</div>
	);
}
