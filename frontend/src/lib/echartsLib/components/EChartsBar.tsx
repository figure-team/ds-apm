import { useCallback } from 'react';

import { buildBarOption } from '../builders/barOption';
import EChartsCartesian, { EChartsCartesianProps } from './EChartsCartesian';

export type EChartsBarProps = Omit<
	EChartsCartesianProps,
	'buildOption' | 'clickMode'
>;

export default function EChartsBar(props: EChartsBarProps): JSX.Element {
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
			buildBarOption({
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
		<EChartsCartesian {...props} buildOption={buildOption} clickMode="bar">
			{props.children}
		</EChartsCartesian>
	);
}
