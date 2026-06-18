import { ToggleGroup, ToggleGroupItem } from '@signozhq/ui';
import { Typography } from 'antd';
import { DisconnectedValuesMode } from 'lib/uPlotV2/config/types';
import { useTranslation } from 'react-i18next';

interface DisconnectValuesModeToggleProps {
	value: DisconnectedValuesMode;
	onChange: (value: DisconnectedValuesMode) => void;
}

export default function DisconnectValuesModeToggle({
	value,
	onChange,
}: DisconnectValuesModeToggleProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<ToggleGroup
			type="single"
			value={value}
			size="lg"
			onChange={(newValue): void => {
				if (newValue) {
					onChange(newValue as DisconnectedValuesMode);
				}
			}}
		>
			<ToggleGroupItem value={DisconnectedValuesMode.Never} aria-label={t('disconnect_values_never')}>
				<Typography.Text className="section-heading-small">{t('disconnect_values_never')}</Typography.Text>
			</ToggleGroupItem>
			<ToggleGroupItem
				value={DisconnectedValuesMode.Threshold}
				aria-label={t('disconnect_values_threshold')}
			>
				<Typography.Text className="section-heading-small">
					{t('disconnect_values_threshold')}
				</Typography.Text>
			</ToggleGroupItem>
		</ToggleGroup>
	);
}
