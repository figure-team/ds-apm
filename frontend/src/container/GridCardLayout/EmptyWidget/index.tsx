import { Typography } from 'antd';
import { useTranslation } from 'react-i18next';

import { Container } from './styles';

function EmptyWidget(): JSX.Element {
	const { t } = useTranslation(['dashboard']);
	return (
		<Container>
			<Typography.Paragraph>{t('empty_widget_prompt')}</Typography.Paragraph>
		</Container>
	);
}

export default EmptyWidget;
