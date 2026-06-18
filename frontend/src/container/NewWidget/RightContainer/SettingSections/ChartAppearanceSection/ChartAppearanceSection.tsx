import { Dispatch, SetStateAction } from 'react';
import { Switch, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import {
	FillMode,
	LineInterpolation,
	LineStyle,
} from 'lib/uPlotV2/config/types';
import { Paintbrush } from 'lucide-react';

import DisconnectValuesSelector from '../../components/DisconnectValuesSelector/DisconnectValuesSelector';
import FillModeSelector from '../../components/FillModeSelector/FillModeSelector';
import LineInterpolationSelector from '../../components/LineInterpolationSelector/LineInterpolationSelector';
import LineStyleSelector from '../../components/LineStyleSelector/LineStyleSelector';
import SettingsSection from '../../components/SettingsSection/SettingsSection';

interface ChartAppearanceSectionProps {
	fillMode: FillMode;
	setFillMode: Dispatch<SetStateAction<FillMode>>;
	lineStyle: LineStyle;
	setLineStyle: Dispatch<SetStateAction<LineStyle>>;
	lineInterpolation: LineInterpolation;
	setLineInterpolation: Dispatch<SetStateAction<LineInterpolation>>;
	showPoints: boolean;
	setShowPoints: Dispatch<SetStateAction<boolean>>;
	spanGaps: boolean | number;
	setSpanGaps: Dispatch<SetStateAction<boolean | number>>;
	allowFillMode: boolean;
	allowLineStyle: boolean;
	allowLineInterpolation: boolean;
	allowShowPoints: boolean;
	allowSpanGaps: boolean;
	stepInterval: number;
}

export default function ChartAppearanceSection({
	fillMode,
	setFillMode,
	lineStyle,
	setLineStyle,
	lineInterpolation,
	setLineInterpolation,
	showPoints,
	setShowPoints,
	spanGaps,
	setSpanGaps,
	allowFillMode,
	allowLineStyle,
	allowLineInterpolation,
	allowShowPoints,
	allowSpanGaps,
	stepInterval,
}: ChartAppearanceSectionProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<SettingsSection title={t('section_chart_appearance')} icon={<Paintbrush size={14} />}>
			{allowFillMode && (
				<FillModeSelector value={fillMode} onChange={setFillMode} />
			)}
			{allowLineStyle && (
				<LineStyleSelector value={lineStyle} onChange={setLineStyle} />
			)}
			{allowLineInterpolation && (
				<LineInterpolationSelector
					value={lineInterpolation}
					onChange={setLineInterpolation}
				/>
			)}
			{allowShowPoints && (
				<section className="show-points toggle-card">
					<div className="toggle-card-text-container">
						<Typography.Text className="section-heading">{t('show_points')}</Typography.Text>
						<Typography.Text className="toggle-card-description">
							{t('show_points_description')}
						</Typography.Text>
					</div>
					<Switch size="small" checked={showPoints} onChange={setShowPoints} />
				</section>
			)}
			{allowSpanGaps && (
				<DisconnectValuesSelector
					value={spanGaps}
					minValue={stepInterval}
					onChange={setSpanGaps}
				/>
			)}
		</SettingsSection>
	);
}
