import { lazy, Suspense, useCallback, useMemo, useRef, useState } from 'react';
import Spinner from 'components/Spinner';
import { PanelWrapperProps } from 'container/PanelWrapper/panelWrapper.types';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useResizeObserver } from 'hooks/useDimensions';
import ChartEngineErrorBoundary from 'lib/echartsLib/components/ChartEngineErrorBoundary';
import { useChartEngine } from 'lib/echartsLib/hooks/useChartEngine';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import uPlot from 'uplot';

import Histogram from '../../charts/Histogram/Histogram';
import ChartManager from '../../components/ChartManager/ChartManager';
import {
	prepareHistogramPanelConfig,
	prepareHistogramPanelData,
} from './utils';

import '../Panel.styles.scss';

// echarts 청크 분리 (스펙 §8 — 컴포넌트 레벨 lazy)
const EChartsHistogram = lazy(
	() => import('lib/echartsLib/components/EChartsHistogram'),
);

function HistogramPanel(props: PanelWrapperProps): JSX.Element {
	const {
		panelMode,
		queryResponse,
		widget,
		isFullViewMode,
		onToggleModelHandler,
	} = props;
	const uPlotRef = useRef<uPlot | null>(null);
	const graphRef = useRef<HTMLDivElement>(null);
	const containerDimensions = useResizeObserver(graphRef);

	const isDarkMode = useIsDarkMode();

	const config = useMemo(() => {
		return prepareHistogramPanelConfig({
			widget,
			isDarkMode,
			apiResponse: queryResponse?.data?.payload,
			panelMode,
		});
	}, [widget, isDarkMode, queryResponse?.data?.payload, panelMode]);

	const chartData = useMemo(() => {
		if (!queryResponse?.data?.payload) {
			return [];
		}
		return prepareHistogramPanelData({
			apiResponse: queryResponse?.data?.payload,
			bucketWidth: widget?.bucketWidth,
			bucketCount: widget?.bucketCount,
			mergeAllActiveQueries: widget?.mergeAllActiveQueries,
		});
	}, [
		queryResponse?.data?.payload,
		widget?.bucketWidth,
		widget?.bucketCount,
		widget?.mergeAllActiveQueries,
	]);

	const layoutChildren = useMemo(() => {
		if (!isFullViewMode || widget.mergeAllActiveQueries) {
			return null;
		}
		return (
			<ChartManager
				config={config}
				alignedData={chartData}
				yAxisUnit={widget.yAxisUnit}
				onCancel={onToggleModelHandler}
			/>
		);
	}, [
		isFullViewMode,
		config,
		chartData,
		widget.yAxisUnit,
		onToggleModelHandler,
		widget.mergeAllActiveQueries,
	]);

	// ECharts 런타임 실패 시 마운트 생애 동안 uPlot 고정 (스펙 §6)
	const [engineFallback, setEngineFallback] = useState(false);
	const handleEngineError = useCallback((error: unknown): void => {
		// eslint-disable-next-line no-console
		console.warn('[HistogramPanel] ECharts 실패 — uPlot 폴백', error);
		setEngineFallback(true);
	}, []);
	const engine = useChartEngine(
		chartData as uPlot.AlignedData,
		widget.chartEngine,
		engineFallback,
	);

	const uPlotBlock = (
		<Histogram
			config={config}
			legendConfig={{
				position: widget?.legendPosition ?? LegendPosition.BOTTOM,
			}}
			plotRef={(plot: uPlot | null): void => {
				uPlotRef.current = plot;
			}}
			onDestroy={(): void => {
				uPlotRef.current = null;
			}}
			canPinTooltip
			yAxisUnit={widget.yAxisUnit}
			decimalPrecision={widget.decimalPrecision}
			isQueriesMerged={widget.mergeAllActiveQueries}
			data={chartData as uPlot.AlignedData}
			width={containerDimensions.width}
			height={containerDimensions.height}
			layoutChildren={layoutChildren}
		/>
	);

	return (
		<div className="panel-container" ref={graphRef}>
			{containerDimensions.width > 0 &&
				containerDimensions.height > 0 &&
				((engine ?? 'uplot') === 'uplot' ? (
					uPlotBlock
				) : (
					<ChartEngineErrorBoundary onError={handleEngineError} fallback={uPlotBlock}>
						<Suspense fallback={<Spinner height="100%" size="large" />}>
							<EChartsHistogram
								widget={widget}
								chartData={chartData as uPlot.AlignedData}
								configBuilder={config}
								apiResponse={queryResponse?.data?.payload}
								currentQuery={widget.query}
								isDarkMode={isDarkMode}
								yAxisUnit={widget.yAxisUnit}
								decimalPrecision={widget.decimalPrecision}
								legendPosition={widget?.legendPosition ?? LegendPosition.BOTTOM}
								isQueriesMerged={widget.mergeAllActiveQueries ?? false}
								canPinTooltip
								width={containerDimensions.width}
								height={containerDimensions.height}
								layoutChildren={layoutChildren}
								onEngineError={handleEngineError}
							/>
						</Suspense>
					</ChartEngineErrorBoundary>
				))}
		</div>
	);
}

export default HistogramPanel;
