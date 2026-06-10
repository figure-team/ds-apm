import { GoogleSquareFilled, KeyOutlined } from '@ant-design/icons';
import { Button, Typography } from 'antd';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

import './CreateEdit.styles.scss';

interface AuthNProvider {
	key: string;
	title: string;
	description: string;
	icon: JSX.Element;
	enabled: boolean;
}

function getAuthNProviders(
	samlEnabled: boolean,
	t: TFunction,
): AuthNProvider[] {
	return [
		{
			key: 'google_auth',
			title: t('selector_google_title'),
			description: t('selector_google_description'),
			icon: <GoogleSquareFilled style={{ fontSize: '37px' }} />,
			enabled: true,
		},
		{
			key: 'saml',
			title: t('selector_saml_title'),
			description: t('selector_saml_description'),
			icon: <KeyOutlined style={{ fontSize: '37px' }} />,
			enabled: samlEnabled,
		},

		{
			key: 'oidc',
			title: t('selector_oidc_title'),
			description: t('selector_oidc_description'),
			icon: <KeyOutlined style={{ fontSize: '37px' }} />,
			enabled: samlEnabled,
		},
	];
}

function AuthnProviderSelector({
	setAuthnProvider,
	samlEnabled,
}: {
	setAuthnProvider: React.Dispatch<React.SetStateAction<string>>;
	samlEnabled: boolean;
}): JSX.Element {
	const { t } = useTranslation('organizationsettings');
	const authnProviders = getAuthNProviders(samlEnabled, t);
	return (
		<div className="authn-provider-selector">
			<section className="header">
				<Typography.Title level={4}>{t('selector_title')}</Typography.Title>
				<Typography.Paragraph italic>{t('selector_subtitle')}</Typography.Paragraph>
			</section>
			<section className="selector">
				{authnProviders.map((provider) => {
					if (provider.enabled) {
						return (
							<section key={provider.key} className="provider">
								<span className="icon">{provider.icon}</span>
								<div className="title-description">
									<Typography.Text className="title">{provider.title}</Typography.Text>
									<Typography.Paragraph className="description">
										{provider.description}
									</Typography.Paragraph>
								</div>
								<Button
									onClick={(): void => setAuthnProvider(provider.key)}
									type="primary"
								>
									{t('configure')}
								</Button>
							</section>
						);
					}
					return <div key={provider.key} />;
				})}
			</section>
		</div>
	);
}

export default AuthnProviderSelector;
