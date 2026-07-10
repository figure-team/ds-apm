import { TrendMetric, TrendSeries } from '../types';

export const TICKS = 4; // Y축 눈금 수(구간)

export interface Scale {
	minT: number;
	maxT: number;
	minV: number;
	maxV: number;
	/** 로그 스케일 하한용 최소 양수값 — 양수 표본이 없으면 undefined */
	minPos?: number;
}

export interface Pad {
	top: number;
	right: number;
	bottom: number;
	left: number;
}

export interface YTick {
	y: number;
	label: string;
}

export interface Mapper {
	x: (tms: number) => number;
	y: (v: number) => number;
	yTicks: () => YTick[];
}

const METRIC_UNIT: Record<TrendMetric, string> = { err: '%', p99: 'ms', rps: '' };

export function formatValue(v: number, metric: TrendMetric): string {
	if (metric === 'err') {
		return v.toFixed(1);
	}
	// 저트래픽 RPS(1 미만)·저지연 ms까지 정수 반올림하면 축 눈금·라벨이 전부
	// "0"으로 뭉개진다(로그 눈금에서 치명적) — 10 미만 구간만 소수 표기로 살린다.
	const abs = Math.abs(v);
	if (abs > 0 && abs < 1) {
		return v.toFixed(2);
	}
	if (abs < 10) {
		return String(Math.round(v * 10) / 10);
	}
	return String(Math.round(v));
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
	const pos = vs.filter((v) => v > 0);
	return {
		minT: Math.min(...ts),
		maxT: Math.max(...ts),
		minV,
		maxV,
		minPos: pos.length > 0 ? Math.min(...pos) : undefined,
	};
}

export function makeMapper(
	scale: Scale,
	w: number,
	h: number,
	pad: Pad,
	metric: TrendMetric,
	logScale: boolean,
): Mapper {
	const innerW = w - pad.left - pad.right;
	const innerH = h - pad.top - pad.bottom;
	const x = (tms: number): number =>
		pad.left + ((tms - scale.minT) / (scale.maxT - scale.minT || 1)) * innerW;

	// 오류율은 0이 유의미한 비율 지표라 로그 제외. 양수 표본이 없으면 로그 불가 → 선형 폴백.
	const useLog =
		logScale &&
		metric !== 'err' &&
		scale.minPos !== undefined &&
		scale.minPos > 0 &&
		scale.maxV > scale.minPos;

	let y: (v: number) => number;
	let tickVal: (k: number) => number;
	if (useLog) {
		const lo = Math.log10(scale.minPos as number);
		const hi = Math.log10(scale.maxV);
		// 0 이하 값은 로그 정의역 밖 → 하한(minPos)으로 클램프해 바닥에 붙인다
		y = (v: number): number =>
			h -
			pad.bottom -
			((Math.log10(Math.max(v, scale.minPos as number)) - lo) / (hi - lo || 1)) *
				innerH;
		tickVal = (k: number): number => 10 ** (lo + ((hi - lo) * k) / TICKS);
	} else {
		y = (v: number): number =>
			h -
			pad.bottom -
			((v - scale.minV) / (scale.maxV - scale.minV || 1)) * innerH;
		tickVal = (k: number): number =>
			scale.minV + ((scale.maxV - scale.minV) * k) / TICKS;
	}
	const yTicks = (): YTick[] =>
		Array.from({ length: TICKS + 1 }, (_, k) => {
			const v = tickVal(k);
			return { y: y(v), label: `${formatValue(v, metric)}${METRIC_UNIT[metric]}` };
		});
	return { x, y, yTicks };
}
