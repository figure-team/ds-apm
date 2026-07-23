import { useEffect, useRef } from 'react';

import echarts, { EChartsOption, EChartsType } from '../echartsCore';
import {
	DSAPM_THEME_DARK,
	DSAPM_THEME_LIGHT,
	registerDsapmThemes,
} from '../themes/dsapmTheme';

export interface EChartsViewProps {
	option: EChartsOption;
	width: number;
	height: number;
	isDarkMode: boolean;
	onError: (error: unknown) => void;
	onInstanceReady?: (chart: EChartsType) => void;
	'data-testid'?: string;
}

export default function EChartsView({
	option,
	width,
	height,
	isDarkMode,
	onError,
	onInstanceReady,
	'data-testid': testId,
}: EChartsViewProps): JSX.Element {
	const containerRef = useRef<HTMLDivElement>(null);
	const chartRef = useRef<EChartsType | null>(null);
	// onError/onInstanceReady가 렌더마다 새 참조여도 인스턴스를 재생성하지 않도록 ref로 고정
	const onErrorRef = useRef(onError);
	onErrorRef.current = onError;
	const onInstanceReadyRef = useRef(onInstanceReady);
	onInstanceReadyRef.current = onInstanceReady;

	// 테마 변경은 인스턴스 재생성이 필요하므로 isDarkMode를 init 의존성에 포함
	useEffect(() => {
		if (!containerRef.current) {
			return undefined;
		}
		try {
			registerDsapmThemes();
			const chart = echarts.init(
				containerRef.current,
				isDarkMode ? DSAPM_THEME_DARK : DSAPM_THEME_LIGHT,
				{ renderer: 'canvas' },
			);
			chartRef.current = chart;
			onInstanceReadyRef.current?.(chart);
		} catch (error) {
			onErrorRef.current(error);
		}
		return (): void => {
			try {
				chartRef.current?.dispose();
			} catch {
				// dispose 실패는 무시 (이미 해제된 인스턴스)
			}
			chartRef.current = null;
		};
	}, [isDarkMode]);

	// 스펙 §6: 명령형 호출은 ErrorBoundary가 못 잡으므로 try/catch로 fail-open
	useEffect(() => {
		if (!chartRef.current) {
			return;
		}
		try {
			// 스펙 §5: replaceMerge로 시리즈 집합 교체 — 유령 시리즈 방지,
			// 동일 id 시리즈는 morphing 트랜지션
			chartRef.current.setOption(option, { replaceMerge: ['series'] });
		} catch (error) {
			onErrorRef.current(error);
		}
	}, [option, isDarkMode]);

	useEffect(() => {
		if (!chartRef.current) {
			return;
		}
		try {
			chartRef.current.resize({ animation: { duration: 0 } });
		} catch (error) {
			onErrorRef.current(error);
		}
	}, [width, height]);

	return (
		<div
			ref={containerRef}
			data-testid={testId}
			style={{ width, height }}
		/>
	);
}
