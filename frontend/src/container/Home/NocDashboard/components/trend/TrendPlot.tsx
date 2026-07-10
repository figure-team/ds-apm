import {
	MouseEvent as ReactMouseEvent,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import { ResolvedTrendSeries, TrendMetric } from '../../types';
import {
	computeScale,
	formatValue,
	makeMapper,
	Pad,
} from '../../utils/trendScale';

// 우측 여백은 끝점 직접 라벨(서비스명+값) 공간, 하단 여백은 X축 시간 라벨 공간
const PAD: Pad = { top: 14, right: 170, bottom: 28, left: 52 };
const LABEL_GAP = 15; // 끝점 라벨 최소 세로 간격(px)
const NAME_MAX = 18; // 라벨 서비스명 최대 길이
const X_TICKS = 4; // X축 시간 눈금 수(구간) → 라벨 5개(양끝 포함)
const SHORT_SPAN_MS = 10 * 60 * 1000; // 이 이하 창이면 초까지 표기

function truncName(name: string): string {
	return name.length > NAME_MAX ? `${name.slice(0, NAME_MAX - 1)}…` : name;
}

function pad2(n: number): string {
	return String(n).padStart(2, '0');
}

// X축 시간 라벨 — 짧은 창(≤10분)은 초까지, 그 외 HH:MM. 로컬 시각(시간 선택기 UTC+9와 일치).
function formatTime(tms: number, spanMs: number): string {
	const d = new Date(tms);
	const hm = `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
	return spanMs <= SHORT_SPAN_MS ? `${hm}:${pad2(d.getSeconds())}` : hm;
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

export interface TrendPlotProps {
	/** 숨김 제외·색 확정이 끝난 표시 대상 계열 */
	series: ResolvedTrendSeries[];
	metric: TrendMetric;
	logScale: boolean;
	thresholdLine?: number;
	hovered: string | null;
}

export default function TrendPlot({
	series,
	metric,
	logScale,
	thresholdLine,
	hovered,
}: TrendPlotProps): JSX.Element {
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
	const mapper = useMemo(
		() => makeMapper(scale, w, h, PAD, metric, logScale),
		[scale, w, h, metric, logScale],
	);

	const drawable = series.filter((s) => !s.missing && s.points.length > 0);
	const labelYs = stackLabelYs(
		drawable.map((s) => mapper.y(s.points[s.points.length - 1].v) + 4),
		PAD.top + 6,
		h - PAD.bottom - 2,
	);

	// ---- 크로스헤어: 표시 계열의 타임스탬프 합집합에 스냅 ----
	const [hoverT, setHoverT] = useState<number | null>(null);
	const timestamps = useMemo(() => {
		const set = new Set<number>();
		drawable.forEach((s) => s.points.forEach((p) => set.add(p.t)));
		return [...set].sort((a, b) => a - b);
	}, [drawable]);
	const valueAt = useMemo(
		() =>
			new Map(
				drawable.map((s) => [s.name, new Map(s.points.map((p) => [p.t, p.v]))]),
			),
		[drawable],
	);

	const handleMove = (e: ReactMouseEvent<HTMLDivElement>): void => {
		const rect = bodyRef.current?.getBoundingClientRect();
		if (!rect || timestamps.length === 0) {
			return;
		}
		const px = e.clientX - rect.left;
		if (px < PAD.left || px > w - PAD.right) {
			setHoverT(null);
			return;
		}
		let best = timestamps[0];
		let bestD = Infinity;
		timestamps.forEach((tt) => {
			const dist = Math.abs(mapper.x(tt) - px);
			if (dist < bestD) {
				bestD = dist;
				best = tt;
			}
		});
		setHoverT(best);
	};

	const rows =
		hoverT === null
			? []
			: drawable
					.map((s) => ({
						name: s.name,
						color: s.resolvedColor,
						v: valueAt.get(s.name)?.get(hoverT),
					}))
					.filter(
						(r): r is { name: string; color: string; v: number } =>
							r.v !== undefined,
					)
					.sort((a, b) => b.v - a.v);
	const cx = hoverT === null ? 0 : mapper.x(hoverT);
	// 우측 거터를 침범하면 왼쪽으로 플립
	const tipLeft = cx + 12 > w - PAD.right - 160 ? cx - 172 : cx + 12;

	return (
		<div
			className="noc-c2-trend-body"
			ref={bodyRef}
			onMouseMove={handleMove}
			onMouseLeave={(): void => setHoverT(null)}
		>
			<svg className="noc-c2-trend-svg" width={w} height={h}>
				{/* Y축 눈금 + 그리드 — 눈금 값·위치는 mapper가 결정(선형/로그 공통) */}
				{mapper.yTicks().map((tk) => (
					<g key={`tick-${tk.label}-${tk.y.toFixed(1)}`} className="noc-c2-tick">
						<line x1={PAD.left} x2={w - PAD.right} y1={tk.y} y2={tk.y} />
						<text x={PAD.left - 8} y={tk.y + 3} textAnchor="end">
							{tk.label}
						</text>
					</g>
				))}
				{/* X축 시간 눈금 + 세로 그리드 (실데이터 있을 때만) */}
				{drawable.length > 0
					? Array.from({ length: X_TICKS + 1 }, (_, k) => {
							const tms = scale.minT + ((scale.maxT - scale.minT) * k) / X_TICKS;
							const tx = mapper.x(tms);
							const anchor = k === 0 ? 'start' : k === X_TICKS ? 'end' : 'middle';
							return (
								<g key={`xtick-${k}`} className="noc-c2-xtick">
									<line x1={tx} x2={tx} y1={PAD.top} y2={h - PAD.bottom} />
									<text x={tx} y={h - PAD.bottom + 16} textAnchor={anchor}>
										{formatTime(tms, scale.maxT - scale.minT)}
									</text>
								</g>
							);
					  })
					: null}
				{metric === 'err' && thresholdLine !== undefined ? (
					<line
						x1={PAD.left}
						x2={w - PAD.right}
						y1={mapper.y(thresholdLine)}
						y2={mapper.y(thresholdLine)}
						className="noc-c2-threshold"
						strokeDasharray="4 4"
					/>
				) : null}
				{drawable.map((s, k) => {
					const d = s.points
						.map(
							(p, j) =>
								`${j === 0 ? 'M' : 'L'}${mapper.x(p.t).toFixed(1)},${mapper
									.y(p.v)
									.toFixed(1)}`,
						)
						.join(' ');
					const last = s.points[s.points.length - 1];
					const dim = hovered && hovered !== s.name;
					return (
						<g key={s.name} opacity={dim ? 0.18 : 1}>
							<path d={d} fill="none" stroke={s.resolvedColor} strokeWidth={2} />
							<circle
								cx={mapper.x(last.t)}
								cy={mapper.y(last.v)}
								r={3}
								fill={s.resolvedColor}
							/>
							{/* 끝점 라벨은 우측 거터에 세로 스택으로 고정 — 플롯 위 뭉침 방지, 거터 활용 */}
							<text
								x={w - PAD.right + 10}
								y={labelYs[k]}
								className="noc-c2-endlabel"
								fill={s.resolvedColor}
							>
								{truncName(s.name)} {formatValue(last.v, metric)}
							</text>
						</g>
					);
				})}
				{hoverT !== null ? (
					<g>
						<line
							className="noc-c2-crosshair"
							x1={cx}
							x2={cx}
							y1={PAD.top}
							y2={h - PAD.bottom}
						/>
						{rows.map((r) => (
							<circle
								key={`hover-${r.name}`}
								cx={cx}
								cy={mapper.y(r.v)}
								r={3.5}
								fill={r.color}
								stroke="var(--noc-panel)"
								strokeWidth={1.5}
							/>
						))}
					</g>
				) : null}
			</svg>
			{hoverT !== null && rows.length > 0 ? (
				<div className="noc-c2-trend-tip" style={{ left: tipLeft, top: 12 }}>
					<div className="noc-c2-tip-time">
						{formatTime(hoverT, scale.maxT - scale.minT)}
					</div>
					{rows.map((r) => (
						<div key={`tip-${r.name}`} className="noc-c2-tip-row">
							<span className="noc-c2-tip-swatch" style={{ background: r.color }} />
							<span>{truncName(r.name)}</span>
							<span className="noc-c2-tip-val">{formatValue(r.v, metric)}</span>
						</div>
					))}
				</div>
			) : null}
		</div>
	);
}
