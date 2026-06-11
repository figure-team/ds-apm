import { useCallback, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Style } from '@signozhq/design-tokens';
import { CircleHelp } from '@signozhq/icons';
import { Callout, Checkbox, Input } from '@signozhq/ui';
import { Form, Tooltip } from 'antd';

import ClaimMappingSection from './components/ClaimMappingSection';
import RoleMappingSection from './components/RoleMappingSection';

import './Providers.styles.scss';

type ExpandedSection = 'claim-mapping' | 'role-mapping' | null;

function ConfigureOIDCAuthnProvider({
	isCreate,
}: {
	isCreate: boolean;
}): JSX.Element {
	const form = Form.useFormInstance();
	const { t } = useTranslation('organizationsettings');

	const [expandedSection, setExpandedSection] = useState<ExpandedSection>(null);

	const handleClaimMappingChange = useCallback((expanded: boolean): void => {
		setExpandedSection(expanded ? 'claim-mapping' : null);
	}, []);

	const handleRoleMappingChange = useCallback((expanded: boolean): void => {
		setExpandedSection(expanded ? 'role-mapping' : null);
	}, []);

	return (
		<div className="authn-provider">
			<section className="authn-provider__header">
				<h3 className="authn-provider__title">{t('oidc_title')}</h3>
				<p className="authn-provider__description">
					<Trans
						t={t}
						i18nKey="oidc_description"
						components={[
							// eslint-disable-next-line jsx-a11y/anchor-has-content
							<a
								key="0"
								href="https://signoz.io/docs/userguide/sso-authentication"
								target="_blank"
								rel="noreferrer"
							/>,
						]}
					/>
				</p>
			</section>

			<div className="authn-provider__columns">
				{/* Left Column - Core OIDC Settings */}
				<div className="authn-provider__left">
					<div className="authn-provider__field-group">
						<label className="authn-provider__label" htmlFor="oidc-domain">
							{t('field_domain')}
							<Tooltip title={t('tooltip_domain')}>
								<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
							</Tooltip>
						</label>
						<Form.Item
							name="name"
							className="authn-provider__form-item"
							rules={[
								{ required: true, message: t('domain_required'), whitespace: true },
							]}
						>
							<Input id="oidc-domain" disabled={!isCreate} />
						</Form.Item>
					</div>

					<div className="authn-provider__field-group">
						<label className="authn-provider__label" htmlFor="oidc-issuer">
							{t('oidc_issuer_url')}
							<Tooltip title={t('oidc_tooltip_issuer_url')}>
								<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
							</Tooltip>
						</label>
						<Form.Item
							name={['oidcConfig', 'issuer']}
							className="authn-provider__form-item"
							rules={[
								{
									required: true,
									message: t('oidc_issuer_url_required'),
									whitespace: true,
								},
							]}
						>
							<Input id="oidc-issuer" />
						</Form.Item>
					</div>

					<div className="authn-provider__field-group">
						<label className="authn-provider__label" htmlFor="oidc-issuer-alias">
							{t('oidc_issuer_alias')}
							<Tooltip title={t('oidc_tooltip_issuer_alias')}>
								<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
							</Tooltip>
						</label>
						<Form.Item
							name={['oidcConfig', 'issuerAlias']}
							className="authn-provider__form-item"
						>
							<Input id="oidc-issuer-alias" />
						</Form.Item>
					</div>

					<div className="authn-provider__field-group">
						<label className="authn-provider__label" htmlFor="oidc-client-id">
							{t('field_client_id')}
							<Tooltip title={t('oidc_tooltip_client_id')}>
								<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
							</Tooltip>
						</label>
						<Form.Item
							name={['oidcConfig', 'clientId']}
							className="authn-provider__form-item"
							rules={[
								{
									required: true,
									message: t('client_id_required'),
									whitespace: true,
								},
							]}
						>
							<Input id="oidc-client-id" />
						</Form.Item>
					</div>

					<div className="authn-provider__field-group">
						<label className="authn-provider__label" htmlFor="oidc-client-secret">
							{t('field_client_secret')}
							<Tooltip title={t('oidc_tooltip_client_secret')}>
								<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
							</Tooltip>
						</label>
						<Form.Item
							name={['oidcConfig', 'clientSecret']}
							className="authn-provider__form-item"
							rules={[
								{
									required: true,
									message: t('client_secret_required'),
									whitespace: true,
								},
							]}
						>
							<Input id="oidc-client-secret" />
						</Form.Item>
					</div>

					<div className="authn-provider__checkbox-row">
						<Form.Item
							name={['oidcConfig', 'insecureSkipEmailVerified']}
							valuePropName="value"
							noStyle
						>
							<Checkbox
								id="oidc-skip-email-verification"
								onChange={(checked: boolean): void => {
									form.setFieldValue(
										['oidcConfig', 'insecureSkipEmailVerified'],
										checked,
									);
								}}
							>
								{t('skip_email_verification')}
							</Checkbox>
						</Form.Item>
						<Tooltip title={t('tooltip_skip_email_verification')}>
							<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
						</Tooltip>
					</div>

					<div className="authn-provider__checkbox-row">
						<Form.Item
							name={['oidcConfig', 'getUserInfo']}
							valuePropName="value"
							noStyle
						>
							<Checkbox
								id="oidc-get-user-info"
								onChange={(checked: boolean): void => {
									form.setFieldValue(['oidcConfig', 'getUserInfo'], checked);
								}}
							>
								{t('oidc_get_user_info')}
							</Checkbox>
						</Form.Item>
						<Tooltip title={t('oidc_tooltip_get_user_info')}>
							<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
						</Tooltip>
					</div>
					<div className="authn-provider__callout-wrapper">
						<Callout type="warning" size="small" showIcon className="callout">
							{t('oidc_callout')}
						</Callout>
					</div>
				</div>

				{/* Right Column - Advanced Settings */}
				<div className="authn-provider__right">
					<ClaimMappingSection
						fieldNamePrefix={['oidcConfig', 'claimMapping']}
						isExpanded={expandedSection === 'claim-mapping'}
						onExpandChange={handleClaimMappingChange}
					/>

					<RoleMappingSection
						fieldNamePrefix={['roleMapping']}
						isExpanded={expandedSection === 'role-mapping'}
						onExpandChange={handleRoleMappingChange}
					/>
				</div>
			</div>
		</div>
	);
}

export default ConfigureOIDCAuthnProvider;
