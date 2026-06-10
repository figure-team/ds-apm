import { InfoCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import {
	DisplayThresholdContainer,
	TypographHeading,
	Typography,
} from './styles';
import { DisplayThresholdProps } from './types';

function DisplayThreshold({ threshold }: DisplayThresholdProps): JSX.Element {
	const { t } = useTranslation(['dashboard']);
	return (
		<DisplayThresholdContainer>
			<TypographHeading>{t('threshold')} </TypographHeading>
			<Typography>{threshold || <InfoCircleOutlined />}</Typography>
		</DisplayThresholdContainer>
	);
}

export default DisplayThreshold;
