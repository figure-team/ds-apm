import { useRef } from 'react';
import type uPlot from 'uplot';

import {
	ChartEngine,
	evaluateChartEngine,
} from '../selectChartEngine';

/**
 * 엔진 판정 결과를 위젯 마운트 생애 동안 유지한다.
 * - 데이터 도착 전에는 null — 호출부는 uPlot 경로를 렌더해 현행(빈 축)을 유지한다.
 *   데이터 도착 후 echarts 판정이면 마운트 교체라 진입 애니메이션은 1회만 재생된다.
 * - forceFallback: 런타임 에러 폴백(스펙 §6) — 한 번이라도 true로 전달되면
 *   전용 래치(fallbackLatchedRef)가 걸려 이후 forceFallback이 다시 false로
 *   돌아와도 마운트 생애 동안 uplot으로 영구 고정된다. 이렇게 하지 않으면
 *   히스테리시스 복귀 경로를 타고 방금 크래시난 echarts 렌더러로 재마운트될 수 있다.
 */
export function useChartEngine(
	data: uPlot.AlignedData | undefined,
	override?: ChartEngine,
	forceFallback?: boolean,
): ChartEngine | null {
	const engineRef = useRef<ChartEngine | undefined>(undefined);
	const fallbackLatchedRef = useRef(false);

	if (forceFallback) {
		fallbackLatchedRef.current = true;
	}

	if (fallbackLatchedRef.current) {
		engineRef.current = 'uplot';
		return 'uplot';
	}

	if (!data || data.length === 0 || (data[0]?.length ?? 0) === 0) {
		return engineRef.current ?? null;
	}

	engineRef.current = evaluateChartEngine({
		data,
		previous: engineRef.current,
		override,
	});
	return engineRef.current;
}
