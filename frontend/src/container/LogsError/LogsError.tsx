import { useTranslation } from 'react-i18next';
import { Typography } from 'antd';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import history from 'lib/history';
import { ArrowRight } from 'lucide-react';

import awwSnapUrl from '@/assets/Icons/awwSnap.svg';

import './LogsError.styles.scss';

export default function LogsError(): JSX.Element {
	const { t } = useTranslation(['logs']);
	const { isCloudUser: isCloudUserVal } = useGetTenantLicense();

	const handleContactSupport = (): void => {
		if (isCloudUserVal) {
			history.push('/support');
		} else {
			window.open('https://signoz.io/slack', '_blank');
		}
	};

	return (
		<div className="logs-error-container">
			<div className="logs-error-content">
				<img src={awwSnapUrl} alt="error-emoji" className="error-state-svg" />
				<Typography.Text>
					<span className="aww-snap">{t('logs:aww_snap')}</span>{' '}
					{t('logs:logs_error_message')}
				</Typography.Text>

				<div className="contact-support" onClick={handleContactSupport}>
					<Typography.Link className="text">
						{t('logs:contact_support')}{' '}
					</Typography.Link>

					<ArrowRight size={14} />
				</div>
			</div>
		</div>
	);
}
