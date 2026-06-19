import { ToggleGroup, ToggleGroupItem } from '@signozhq/ui';
import { Typography } from 'antd';
import { LineStyle } from 'lib/uPlotV2/config/types';
import { useTranslation } from 'react-i18next';

import './LineStyleSelector.styles.scss';

interface LineStyleSelectorProps {
	value: LineStyle;
	onChange: (value: LineStyle) => void;
}

export default function LineStyleSelector({
	value,
	onChange,
}: LineStyleSelectorProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<section className="line-style-selector control-container">
			<Typography.Text className="section-heading">{t('line_style')}</Typography.Text>
			<ToggleGroup
				type="single"
				value={value}
				size="lg"
				onChange={(newValue): void => {
					if (newValue) {
						onChange(newValue as LineStyle);
					}
				}}
			>
				<ToggleGroupItem value={LineStyle.Solid} aria-label={t('line_style_solid')}>
					<svg
						className="line-style-icon"
						viewBox="0 0 48 48"
						fill="none"
						stroke="#888"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
					>
						<path d="M8 24 L40 24" />
					</svg>
					<Typography.Text className="section-heading-small">{t('line_style_solid')}</Typography.Text>
				</ToggleGroupItem>
				<ToggleGroupItem value={LineStyle.Dashed} aria-label={t('line_style_dashed')}>
					<svg
						className="line-style-icon"
						viewBox="0 0 48 48"
						fill="none"
						stroke="#888"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeDasharray="6 4"
					>
						<path d="M8 24 L40 24" />
					</svg>
					<Typography.Text className="section-heading-small">{t('line_style_dashed')}</Typography.Text>
				</ToggleGroupItem>
			</ToggleGroup>
		</section>
	);
}
