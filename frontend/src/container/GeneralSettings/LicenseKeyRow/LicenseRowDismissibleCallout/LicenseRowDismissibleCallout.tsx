import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Callout } from '@signozhq/ui';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import { FeatureKeys } from 'constants/features';
import { LOCALSTORAGE } from 'constants/localStorage';
import ROUTES from 'constants/routes';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import './LicenseRowDismissible.styles.scss';

function LicenseRowDismissibleCallout(): JSX.Element | null {
	const { t } = useTranslation(['generalSettings']);
	const [isCalloutDismissed, setIsCalloutDismissed] = useState<boolean>(
		() =>
			getLocalStorageApi(LOCALSTORAGE.LICENSE_KEY_CALLOUT_DISMISSED) === 'true',
	);

	const { user, featureFlags } = useAppContext();
	const { isCloudUser } = useGetTenantLicense();

	const isAdmin = user.role === USER_ROLES.ADMIN;
	const isEditor = user.role === USER_ROLES.EDITOR;

	const isGatewayEnabled =
		featureFlags?.find((feature) => feature.name === FeatureKeys.GATEWAY)
			?.active || false;

	const hasServiceAccountsAccess = isAdmin;

	const hasIngestionAccess =
		(isCloudUser && !isGatewayEnabled) ||
		(isGatewayEnabled && (isAdmin || isEditor));

	const handleDismissCallout = (): void => {
		setLocalStorageApi(LOCALSTORAGE.LICENSE_KEY_CALLOUT_DISMISSED, 'true');
		setIsCalloutDismissed(true);
	};

	return !isCalloutDismissed ? (
		<Callout
			type="info"
			size="small"
			showIcon
			action="dismissible"
			onClick={handleDismissCallout}
			className="license-key-callout"
		>
			<div className="license-key-callout__description">
				<Trans
					t={t}
					i18nKey="callout_not_your_key"
					components={[<strong key="0" />]}
				/>
				{(hasServiceAccountsAccess || hasIngestionAccess) && (
					<>
						{' '}
						{t('callout_find_your')}{' '}
						{hasServiceAccountsAccess && (
							<Link
								to={ROUTES.SERVICE_ACCOUNTS_SETTINGS}
								className="license-key-callout__link"
							>
								{t('callout_service_account_link')}
							</Link>
						)}
						{hasServiceAccountsAccess && hasIngestionAccess && ` ${t('and')} `}
						{hasIngestionAccess && (
							<Link
								to={ROUTES.INGESTION_SETTINGS}
								className="license-key-callout__link"
							>
								{t('callout_ingestion_link')}
							</Link>
						)}
						.
					</>
				)}
			</div>
		</Callout>
	) : null;
}

export default LicenseRowDismissibleCallout;
