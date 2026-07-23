import type uPlot from 'uplot';

export type ChartEngine = 'echarts' | 'uplot';

// 스펙 §3.1 — 초안 임계치. 실측(Task 12) 후 보정 가능
export const ENGINE_LIMITS = {
	maxSeries: 30,
	maxTotalPoints: 50000,
	recoveryRatio: 0.8,
} as const;

export function evaluateChartEngine({
	data,
	previous,
	override,
}: {
	data: uPlot.AlignedData;
	previous?: ChartEngine;
	override?: ChartEngine;
}): ChartEngine {
	// 위젯 단위 오버라이드는 크기 안전캡을 우회한다(명시 'echarts'면 100시리즈여도
	// 강등 안 함) — 의도된 시맨틱. 단, 런타임 에러 폴백 래치(useChartEngine)는 이보다
	// 상위에서 동작하므로 오버라이드가 크래시 재시도를 유발하지는 않는다.
	if (override) {
		return override;
	}

	const seriesCount = Math.max(0, data.length - 1);
	const pointCount = data[0]?.length ?? 0;
	const totalPoints = seriesCount * pointCount;

	const isOverLimit =
		seriesCount > ENGINE_LIMITS.maxSeries ||
		totalPoints > ENGINE_LIMITS.maxTotalPoints;

	if (isOverLimit) {
		return 'uplot';
	}

	// 히스테리시스: 강등 상태에서는 임계의 80% 미만으로 내려와야 복귀
	// (자동 리프레시로 경계를 오갈 때 엔진 리마운트 플래핑 방지 — 스펙 §3.1)
	if (previous === 'uplot') {
		const canRecover =
			seriesCount < ENGINE_LIMITS.maxSeries * ENGINE_LIMITS.recoveryRatio &&
			totalPoints < ENGINE_LIMITS.maxTotalPoints * ENGINE_LIMITS.recoveryRatio;
		return canRecover ? 'echarts' : 'uplot';
	}

	return 'echarts';
}
