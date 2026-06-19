import { Tag } from 'antd';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useTranslation } from 'react-i18next';

function Severity({ severity }: SeverityProps): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { t } = useTranslation('alerts');

	switch (severity) {
		case 'unprocessed': {
			return <Tag color={isDarkMode ? 'green' : '#16A34A'}>{t('triggered_unprocessed')}</Tag>;
		}

		case 'active': {
			return <Tag color={isDarkMode ? 'red' : '#DC2626'}>{t('triggered_firing')}</Tag>;
		}

		case 'suppressed': {
			return <Tag color={isDarkMode ? 'red' : '#DC2626'}>{t('triggered_suppressed')}</Tag>;
		}

		default: {
			return <Tag color="default">{t('triggered_unknown')}</Tag>;
		}
	}
}

interface SeverityProps {
	severity: string;
}

export default Severity;
