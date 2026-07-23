import {
	lazy,
	Suspense,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';
import Spinner from 'components/Spinner';
import { PanelWrapperProps } from 'container/PanelWrapper/panelWrapper.types';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useResizeObserver } from 'hooks/useDimensions';
import ChartEngineErrorBoundary from 'lib/echartsLib/components/ChartEngineErrorBoundary';
import { useChartEngine } from 'lib/echartsLib/hooks/useChartEngine';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import ContextMenu from 'periscope/components/ContextMenu';
import { useTimezone } from 'providers/Timezone';
import uPlot from 'uplot';
import { getTimeRange } from 'utils/getTimeRange';

import BarChart from '../../charts/BarChart/BarChart';
import ChartManager from '../../components/ChartManager/ChartManager';
import { usePanelContextMenu } from '../../hooks/usePanelContextMenu';
import { prepareBarPanelConfig, prepareBarPanelData } from './utils';

import '../Panel.styles.scss';

// echarts 청크 분리 (스펙 §8 — 컴포넌트 레벨 lazy)
const EChartsBar = lazy(() => import('lib/echartsLib/components/EChartsBar'));

function BarPanel(props: PanelWrapperProps): JSX.Element {
	const {
		panelMode,
		queryResponse,
		widget,
		onDragSelect,
		isFullViewMode,
		onToggleModelHandler,
	} = props;
	const uPlotRef = useRef<uPlot | null>(null);
	const graphRef = useRef<HTMLDivElement>(null);
	const [minTimeScale, setMinTimeScale] = useState<number>();
	const [maxTimeScale, setMaxTimeScale] = useState<number>();
	const containerDimensions = useResizeObserver(graphRef);

	const isDarkMode = useIsDarkMode();
	const { timezone } = useTimezone();

	useEffect((): void => {
		const { startTime, endTime } = getTimeRange(queryResponse);

		setMinTimeScale(startTime);
		setMaxTimeScale(endTime);
	}, [queryResponse]);

	const {
		coordinates,
		popoverPosition,
		onClose,
		menuItemsConfig,
		clickHandlerWithContextMenu,
	} = usePanelContextMenu({
		widget,
		queryResponse,
	});

	const config = useMemo(() => {
		return prepareBarPanelConfig({
			widget,
			isDarkMode,
			currentQuery: widget.query,
			onClick: clickHandlerWithContextMenu,
			onDragSelect,
			apiResponse: queryResponse?.data?.payload,
			timezone,
			panelMode,
			minTimeScale: minTimeScale,
			maxTimeScale: maxTimeScale,
		});
	}, [
		widget,
		isDarkMode,
		queryResponse?.data?.payload,
		clickHandlerWithContextMenu,
		onDragSelect,
		minTimeScale,
		maxTimeScale,
		timezone,
		panelMode,
	]);

	const chartData = useMemo(() => {
		if (!queryResponse?.data?.payload) {
			return [];
		}
		return prepareBarPanelData(queryResponse?.data?.payload);
	}, [queryResponse?.data?.payload]);

	const layoutChildren = useMemo(() => {
		if (!isFullViewMode) {
			return null;
		}
		return (
			<ChartManager
				config={config}
				alignedData={chartData}
				yAxisUnit={widget.yAxisUnit}
				decimalPrecision={widget.decimalPrecision}
				onCancel={onToggleModelHandler}
			/>
		);
	}, [
		isFullViewMode,
		config,
		chartData,
		widget.yAxisUnit,
		onToggleModelHandler,
		widget.decimalPrecision,
	]);

	const onPlotDestroy = useCallback(() => {
		uPlotRef.current = null;
	}, []);

	const onPlotRef = useCallback((plot: uPlot | null): void => {
		uPlotRef.current = plot;
	}, []);

	// ECharts 런타임 실패 시 마운트 생애 동안 uPlot 고정 (스펙 §6)
	const [engineFallback, setEngineFallback] = useState(false);
	const handleEngineError = useCallback((error: unknown): void => {
		// eslint-disable-next-line no-console
		console.warn('[BarPanel] ECharts 실패 — uPlot 폴백', error);
		setEngineFallback(true);
	}, []);
	const engine = useChartEngine(
		chartData as uPlot.AlignedData,
		widget.chartEngine,
		engineFallback,
	);

	const uPlotBlock = (
		<BarChart
			config={config}
			legendConfig={{
				position: widget?.legendPosition ?? LegendPosition.BOTTOM,
			}}
			canPinTooltip
			plotRef={onPlotRef}
			onDestroy={onPlotDestroy}
			data={chartData as uPlot.AlignedData}
			width={containerDimensions.width}
			height={containerDimensions.height}
			layoutChildren={layoutChildren}
			isStackedBarChart={widget.stackedBarChart ?? false}
			yAxisUnit={widget.yAxisUnit}
			decimalPrecision={widget.decimalPrecision}
			timezone={timezone}
		>
			<ContextMenu
				coordinates={coordinates}
				popoverPosition={popoverPosition}
				title={menuItemsConfig.header as string}
				items={menuItemsConfig.items}
				onClose={onClose}
			/>
		</BarChart>
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
							<EChartsBar
								widget={widget}
								chartData={chartData as uPlot.AlignedData}
								configBuilder={config}
								apiResponse={queryResponse?.data?.payload}
								currentQuery={widget.query}
								isDarkMode={isDarkMode}
								timezone={timezone}
								yAxisUnit={widget.yAxisUnit}
								decimalPrecision={widget.decimalPrecision}
								legendPosition={widget?.legendPosition ?? LegendPosition.BOTTOM}
								minTimeScale={minTimeScale}
								maxTimeScale={maxTimeScale}
								onDragSelect={onDragSelect}
								onClick={clickHandlerWithContextMenu}
								canPinTooltip
								width={containerDimensions.width}
								height={containerDimensions.height}
								layoutChildren={layoutChildren}
								onEngineError={handleEngineError}
							>
								<ContextMenu
									coordinates={coordinates}
									popoverPosition={popoverPosition}
									title={menuItemsConfig.header as string}
									items={menuItemsConfig.items}
									onClose={onClose}
								/>
							</EChartsBar>
						</Suspense>
					</ChartEngineErrorBoundary>
				))}
		</div>
	);
}

export default BarPanel;
