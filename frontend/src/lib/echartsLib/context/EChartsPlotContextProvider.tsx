import { PropsWithChildren, useCallback, useMemo, useRef } from 'react';
import type { SeriesVisibilityItem } from 'container/DashboardContainer/visualization/panels/types';
import { updateSeriesVisibilityToLocalStorage } from 'container/DashboardContainer/visualization/panels/utils/legendVisibilityUtils';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import { IPlotContext, PlotContext } from 'lib/uPlotV2/context/PlotContext';

import { EChartsType } from '../echartsCore';

interface Props {
	chart: EChartsType | null;
	widgetId: string;
	seriesLabels: string[];
	/** 범례 계약용 병행 config — setSeries 훅 수동 발화 대상 (리뷰 반영) */
	config: UPlotConfigBuilder;
	/** 표시 상태 변경 통지 — Task 10이 option 재빌드·심 show에 사용 (리뷰 반영) */
	onVisibilityChange?: (map: Record<number, boolean>) => void;
	shouldSaveSelectionPreference?: boolean;
}

/**
 * 기존 Legend/ChartManager가 쓰는 IPlotContext의 ECharts 구현.
 * 인덱스 규약은 uPlot(0=x축, 시리즈는 1부터)을 따른다 — 스펙 §3.3.
 *
 * 리뷰 반영 — 표시 상태의 단일 소스는 visibilityRef다.
 * Legend UI는 useLegendsSync가 config.addHook('setSeries')로만 갱신되므로
 * 토글 시 getConfig().hooks.setSeries 배열을 수동 발화한다.
 * getConfig()는 훅 배열을 참조로 반환하고(UPlotConfigBuilder.ts:453),
 * 핸들러(useLegendsSync.handleSetSeries)는 첫 인자 u를 사용하지 않아
 * null 전달이 안전하다.
 */
export default function EChartsPlotContextProvider({
	chart,
	widgetId,
	seriesLabels,
	config,
	onVisibilityChange,
	shouldSaveSelectionPreference = false,
	children,
}: PropsWithChildren<Props>): JSX.Element {
	const soloIndexRef = useRef<number | undefined>(undefined);
	const chartRef = useRef(chart);
	chartRef.current = chart;
	const labelsRef = useRef(seriesLabels);
	labelsRef.current = seriesLabels;
	const configRef = useRef(config);
	configRef.current = config;
	// seriesIndex(1..n) → show. 미기록은 true 취급
	const visibilityRef = useRef<Record<number, boolean>>({});
	const onVisibilityChangeRef = useRef(onVisibilityChange);
	onVisibilityChangeRef.current = onVisibilityChange;

	const labelOf = (seriesIndex: number): string | undefined =>
		labelsRef.current[seriesIndex - 1];

	const applyVisibility = useCallback((next: Record<number, boolean>): void => {
		visibilityRef.current = next;
		// Legend UI 동기화 — setSeries 훅 수동 발화 (계약: 변경된 전체 map을 순회해 발화)
		const hooks = configRef.current.getConfig().hooks?.setSeries ?? [];
		Object.entries(next).forEach(([idx, show]) => {
			hooks.forEach((fn) => fn?.(null as never, Number(idx), { show } as never));
		});
		// 차트 반영은 option 재빌드로 (Task 10 — 숨김 시리즈 제외 + replaceMerge)
		onVisibilityChangeRef.current?.({ ...next });
	}, []);

	const syncSeriesVisibilityToLocalStorage = useCallback((): void => {
		if (!widgetId) {
			return;
		}
		const items: SeriesVisibilityItem[] = [
			{ label: 'Timestamp', show: true }, // 저장 포맷 규약: [0]=x축 자리
			...labelsRef.current.map((label, i) => ({
				label,
				show: visibilityRef.current[i + 1] ?? true,
			})),
		];
		updateSeriesVisibilityToLocalStorage(widgetId, items);
	}, [widgetId]);

	const maybeSave = useCallback((): void => {
		if (shouldSaveSelectionPreference) {
			syncSeriesVisibilityToLocalStorage();
		}
	}, [shouldSaveSelectionPreference, syncSeriesVisibilityToLocalStorage]);

	const onToggleSeriesOnOff = useCallback(
		(seriesIndex: number): void => {
			if (!labelOf(seriesIndex)) {
				return;
			}
			const next = { ...visibilityRef.current };
			next[seriesIndex] = !(next[seriesIndex] ?? true);
			applyVisibility(next);
			maybeSave();
		},
		[applyVisibility, maybeSave],
	);

	const onToggleSeriesVisibility = useCallback(
		(seriesIndex: number): void => {
			if (!labelOf(seriesIndex)) {
				return;
			}
			// uPlot 경로와 동일한 솔로 토글: 같은 시리즈 재클릭 시 전체 복원
			const isReset = soloIndexRef.current === seriesIndex;
			soloIndexRef.current = isReset ? undefined : seriesIndex;
			const next: Record<number, boolean> = {};
			labelsRef.current.forEach((_, i) => {
				next[i + 1] = isReset || i + 1 === seriesIndex;
			});
			applyVisibility(next);
			maybeSave();
		},
		[applyVisibility, maybeSave],
	);

	const onFocusSeries = useCallback((seriesIndex: number | null): void => {
		const instance = chartRef.current;
		if (!instance) {
			return;
		}
		if (seriesIndex === null) {
			instance.dispatchAction({ type: 'downplay' });
			return;
		}
		const name = labelOf(seriesIndex);
		if (name) {
			instance.dispatchAction({ type: 'highlight', seriesName: name });
		}
	}, []);

	// uPlot 전용 초기화 훅 — ECharts 경로에서는 no-op (실 시그니처 인자는 무시)
	const setPlotContextInitialState = useCallback((): void => undefined, []);

	const value = useMemo<IPlotContext>(
		() => ({
			setPlotContextInitialState,
			onToggleSeriesVisibility,
			onToggleSeriesOnOff,
			onFocusSeries,
			syncSeriesVisibilityToLocalStorage,
		}),
		[
			setPlotContextInitialState,
			onToggleSeriesVisibility,
			onToggleSeriesOnOff,
			onFocusSeries,
			syncSeriesVisibilityToLocalStorage,
		],
	);

	return <PlotContext.Provider value={value}>{children}</PlotContext.Provider>;
}
