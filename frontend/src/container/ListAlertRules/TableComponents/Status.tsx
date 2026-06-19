import { Tag } from 'antd';
import type { RuletypesRuleDTO } from 'api/generated/services/sigNoz.schemas';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useTranslation } from 'react-i18next';

function Status({ status }: StatusProps): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { t } = useTranslation('alerts');

	switch (status) {
		case 'inactive': {
			return <Tag color={isDarkMode ? 'green' : '#16A34A'}>{t('status_ok')}</Tag>;
		}

		case 'pending': {
			return <Tag color={isDarkMode ? 'orange' : '#F59E0B'}>{t('status_pending')}</Tag>;
		}

		case 'firing': {
			return <Tag color={isDarkMode ? 'red' : '#DC2626'}>{t('status_firing')}</Tag>;
		}

		case 'disabled': {
			return <Tag>{t('status_disabled')}</Tag>;
		}

		default: {
			return <Tag color="default">{t('status_unknown')}</Tag>;
		}
	}
}

interface StatusProps {
	status: RuletypesRuleDTO['state'];
}

export default Status;
