import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import RouteTab from 'components/RouteTab';
import { FeatureKeys } from 'constants/features';
import useComponentPermission from 'hooks/useComponentPermission';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import history from 'lib/history';
import { Cog } from 'lucide-react';
import { useAppContext } from 'providers/App/App';

import { getRoutes } from './utils';

import './Settings.styles.scss';

function SettingsPage(): JSX.Element {
	const { user, featureFlags, trialInfo } = useAppContext();
	const { isCloudUser, isEnterpriseSelfHostedUser } = useGetTenantLicense();

	const isWorkspaceBlocked = trialInfo?.workSpaceBlock || false;

	const [isCurrentOrgSettings] = useComponentPermission(
		['current_org_settings'],
		user.role,
	);
	const { t } = useTranslation(['routes']);

	const isGatewayEnabled =
		featureFlags?.find((feature) => feature.name === FeatureKeys.GATEWAY)
			?.active || false;

	const routes = useMemo(
		() =>
			getRoutes(
				user.role,
				isCurrentOrgSettings,
				isGatewayEnabled,
				isWorkspaceBlocked,
				isCloudUser,
				isEnterpriseSelfHostedUser,
				t,
			),
		[
			user.role,
			isCurrentOrgSettings,
			isGatewayEnabled,
			isWorkspaceBlocked,
			isCloudUser,
			isEnterpriseSelfHostedUser,
			t,
		],
	);

	return (
		<div className="settings-page">
			<header className="settings-page-header">
				<div
					className="settings-page-header-title"
					data-testid="settings-page-title"
				>
					<Cog size={16} />
					{t('routes:settings_title')}
				</div>
			</header>

			<div className="settings-page-content-container">
				<div className="settings-page-content">
					<RouteTab
						routes={routes}
						activeKey={history.location.pathname}
						history={history}
						tabBarStyle={{ display: 'none' }}
					/>
				</div>
			</div>
		</div>
	);
}

export default SettingsPage;
