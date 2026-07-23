import { themeColors } from 'constants/theme';
import { generateColor } from 'lib/uPlotLib/utils/generateColor';

import echarts from '../echartsCore';

// 목업 게이트(Task 1)에서 승인된 시안 파라미터 — 시안 A 기본값
export const MOCKUP_TUNING = {
	areaGradient: true,
	areaAlphaTop: '55', // 그라데이션 상단 알파 (hex 2자리)
	lineWidth: 2,
	entryDurationMs: 600,
	updateDurationMs: 300,
	emphasisLineWidth: 3.5,
	blurOpacity: 0.2,
} as const;

// 막대 시안 A 채택 파라미터 (목업 게이트 2026-07-23) — 라인 MOCKUP_TUNING과 같은 위치에 모음
export const BAR_MOCKUP_TUNING = {
	areaAlphaTop: 'ff', // 그라데이션 상단(불투명)
	areaAlphaBottom: '55', // 그라데이션 하단
	borderRadius: 4, // 상단 모서리 반경(px)
	barMaxWidth: 22,
	entryDurationMs: 600,
	updateDurationMs: 300,
	staggerMs: 6, // 포인트 인덱스당 진입 지연
	seriesStaggerMs: 80, // 시리즈당 진입 지연
	blurOpacity: 0.25,
} as const;

export const DSAPM_THEME_DARK = 'dsapm-dark';
export const DSAPM_THEME_LIGHT = 'dsapm-light';

/**
 * uPlot 경로(UPlotSeriesBuilder)와 동일한 색 결정 우선순위:
 * colorMapping[label] → generateColor(label, 모드별 팔레트)
 * 같은 시리즈는 렌더러와 무관하게 같은 색이어야 한다 (스펙 §3.2)
 */
export function getSeriesColor(
	label: string,
	colorMapping: Record<string, string>,
	isDarkMode: boolean,
): string {
	return (
		colorMapping[label] ??
		generateColor(
			label,
			isDarkMode ? themeColors.chartcolors : themeColors.lightModeColor,
		)
	);
}

export function buildDsapmTheme(isDarkMode: boolean): Record<string, unknown> {
	const axisLabelColor = isDarkMode ? '#9aa0a6' : '#5f6570';
	const axisLineColor = isDarkMode ? '#3c3f45' : '#d5d8de';
	const splitLineColor = isDarkMode ? '#26282c' : '#eceef2';

	const axis = {
		axisLine: { lineStyle: { color: axisLineColor } },
		axisTick: { show: false },
		axisLabel: { color: axisLabelColor, fontSize: 11 },
		splitLine: { lineStyle: { color: splitLineColor } },
	};

	return {
		backgroundColor: 'transparent',
		textStyle: { fontFamily: 'inherit' },
		categoryAxis: axis,
		valueAxis: axis,
		timeAxis: axis,
	};
}

let registered = false;
export function registerDsapmThemes(): void {
	if (registered) {
		return;
	}
	echarts.registerTheme(DSAPM_THEME_DARK, buildDsapmTheme(true));
	echarts.registerTheme(DSAPM_THEME_LIGHT, buildDsapmTheme(false));
	registered = true;
}
