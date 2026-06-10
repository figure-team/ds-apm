import { useTranslation } from 'react-i18next';
import { useCopyToClipboard } from 'react-use';
import { Button } from '@signozhq/ui';
import { Typography } from 'antd';
import { useNotifications } from 'hooks/useNotifications';
import { Copy } from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { getMaskedKey } from 'utils/maskedKey';

import './LicenseSection.styles.scss';

function LicenseSection(): JSX.Element | null {
	const { t } = useTranslation(['settings']);
	const { activeLicense } = useAppContext();
	const { notifications } = useNotifications();
	const [, handleCopyToClipboard] = useCopyToClipboard();

	const handleCopyKey = (text: string): void => {
		handleCopyToClipboard(text);
		notifications.success({
			message: t('settings:copied_to_clipboard'),
		});
	};

	if (!activeLicense?.key) {
		return <></>;
	}

	return (
		<div className="license-section">
			<div className="license-section-header">
				<div className="license-section-title">{t('settings:license')}</div>
			</div>

			<div className="license-section-content">
				<div className="license-section-content-item">
					<div className="license-section-content-item-title-action">
						<span>{t('settings:license_key')}</span>
						<span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
							<Typography.Text code>{getMaskedKey(activeLicense.key)}</Typography.Text>
							<Button
								variant="link"
								color="none"
								aria-label={t('settings:copy_license_key')}
								data-testid="license-key-copy-btn"
								onClick={(): void => handleCopyKey(activeLicense.key)}
							>
								<Copy size={14} />
							</Button>
						</span>
					</div>

					<div className="license-section-content-item-description">
						{t('settings:license_key_description')}
					</div>
				</div>
			</div>
		</div>
	);
}

export default LicenseSection;
