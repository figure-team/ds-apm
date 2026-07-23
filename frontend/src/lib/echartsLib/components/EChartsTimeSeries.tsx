import { useCallback } from 'react';
import { Timezone } from 'components/CustomTimePicker/timezoneUtils';
import { PrecisionOption } from 'components/Graph/types';
import { OnClickPluginOpts } from 'lib/uPlotLib/plugins/onClickPlugin';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import type uPlot from 'uplot';
import { Widgets } from 'types/api/dashboard/getAll';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import { buildTimeSeriesOption } from '../builders/timeSeriesOption';
import EChartsCartesian from './EChartsCartesian';

export interface EChartsTimeSeriesProps {
	widget: Widgets;
	chartData: uPlot.AlignedData;
	configBuilder: UPlotConfigBuilder;
	apiResponse?: MetricRangePayloadProps;
	currentQuery?: Query;
	isDarkMode: boolean;
	timezone: Timezone;
	yAxisUnit?: string;
	decimalPrecision?: PrecisionOption;
	legendPosition: LegendPosition;
	minTimeScale?: number;
	maxTimeScale?: number;
	onDragSelect: (start: number, end: number) => void;
	onClick?: OnClickPluginOpts['onClick'];
	canPinTooltip?: boolean;
	width: number;
	height: number;
	layoutChildren?: React.ReactNode;
	children?: React.ReactNode;
	onEngineError: (error: unknown) => void;
}

export default function EChartsTimeSeries(
	props: EChartsTimeSeriesProps,
): JSX.Element {
	const {
		widget,
		apiResponse,
		chartData,
		currentQuery,
		isDarkMode,
		minTimeScale,
		maxTimeScale,
		timezone,
	} = props;

	const buildOption = useCallback(
		({
			visibilityMap,
			reducedMotion,
		}: {
			visibilityMap: Record<number, boolean>;
			reducedMotion: boolean;
		}) =>
			buildTimeSeriesOption({
				widget,
				apiResponse,
				chartData,
				currentQuery,
				isDarkMode,
				reducedMotion,
				minTimeScale,
				maxTimeScale,
				timezone,
				visibilityMap,
			}),
		[
			widget,
			apiResponse,
			chartData,
			currentQuery,
			isDarkMode,
			minTimeScale,
			maxTimeScale,
			timezone,
		],
	);

	return (
		<EChartsCartesian {...props} buildOption={buildOption} clickMode="line">
			{props.children}
		</EChartsCartesian>
	);
}
