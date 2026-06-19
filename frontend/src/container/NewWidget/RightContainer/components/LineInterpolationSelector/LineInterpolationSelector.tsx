import { ToggleGroup, ToggleGroupItem } from '@signozhq/ui';
import { Typography } from 'antd';
import { LineInterpolation } from 'lib/uPlotV2/config/types';
import { useTranslation } from 'react-i18next';

import './LineInterpolationSelector.styles.scss';

interface LineInterpolationSelectorProps {
	value: LineInterpolation;
	onChange: (value: LineInterpolation) => void;
}

export default function LineInterpolationSelector({
	value,
	onChange,
}: LineInterpolationSelectorProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<section className="line-interpolation-selector control-container">
			<Typography.Text className="section-heading">
				{t('line_interpolation')}
			</Typography.Text>
			<ToggleGroup
				type="single"
				value={value}
				size="lg"
				onChange={(newValue): void => {
					if (newValue) {
						onChange(newValue as LineInterpolation);
					}
				}}
			>
				<ToggleGroupItem value={LineInterpolation.Linear} aria-label={t('line_interpolation_linear')}>
					<svg
						className="line-interpolation-icon"
						viewBox="0 0 48 48"
						fill="none"
						stroke="#888"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
					>
						<circle cx="8" cy="32" r="3" fill="#888" />
						<circle cx="24" cy="16" r="3" fill="#888" />
						<circle cx="40" cy="32" r="3" fill="#888" />
						<path d="M8 32 L24 16 L40 32" stroke="#888" />
					</svg>
				</ToggleGroupItem>
				<ToggleGroupItem value={LineInterpolation.Spline} aria-label={t('line_interpolation_spline')}>
					<svg
						className="line-interpolation-icon"
						viewBox="0 0 48 48"
						fill="none"
						stroke="#888"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
					>
						<circle cx="8" cy="32" r="3" fill="#888" />
						<circle cx="24" cy="16" r="3" fill="#888" />
						<circle cx="40" cy="32" r="3" fill="#888" />
						<path d="M8 32 C16 8, 32 8, 40 32" />
					</svg>
				</ToggleGroupItem>
				<ToggleGroupItem
					value={LineInterpolation.StepAfter}
					aria-label={t('line_interpolation_step_after')}
				>
					<svg
						className="line-interpolation-icon"
						viewBox="0 0 48 48"
						fill="none"
						stroke="#888"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
					>
						<circle cx="8" cy="32" r="3" fill="#888" />
						<circle cx="24" cy="16" r="3" fill="#888" />
						<circle cx="40" cy="32" r="3" fill="#888" />
						<path d="M8 32 V16 H24 V32 H40" />
					</svg>
				</ToggleGroupItem>

				<ToggleGroupItem
					value={LineInterpolation.StepBefore}
					aria-label={t('line_interpolation_step_before')}
				>
					<svg
						className="line-interpolation-icon"
						viewBox="0 0 48 48"
						fill="none"
						stroke="#888"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
					>
						<circle cx="8" cy="32" r="3" fill="#888" />
						<circle cx="24" cy="16" r="3" fill="#888" />
						<circle cx="40" cy="32" r="3" fill="#888" />
						<path d="M8 32 H24 V16 H40 V32" />
					</svg>
				</ToggleGroupItem>
			</ToggleGroup>
		</section>
	);
}
