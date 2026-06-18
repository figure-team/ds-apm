import { useTranslation } from 'react-i18next';
import { Button, Tabs, Tooltip } from 'antd';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import { Braces, Globe, Table } from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import { VariablesSettingsTabHandle } from '../DashboardDescription/types';
import DashboardVariableSettings from './DashboardVariableSettings';
import GeneralDashboardSettings from './General';
import PublicDashboardSetting from './PublicDashboard';

import './DashboardSettingsContent.styles.scss';

function DashboardSettings({
	variablesSettingsTabHandle,
}: {
	variablesSettingsTabHandle: VariablesSettingsTabHandle;
}): JSX.Element {
	const { t } = useTranslation('dashboard');
	const { user } = useAppContext();
	const { isCloudUser, isEnterpriseSelfHostedUser } = useGetTenantLicense();

	const enablePublicDashboard = isCloudUser || isEnterpriseSelfHostedUser;

	const publicDashboardItem = {
		label: (
			<Tooltip
				title={
					user?.role !== USER_ROLES.ADMIN
						? 'Only admins can publish / manage public dashboards'
						: ''
				}
				placement="right"
			>
				<Button
					type="text"
					icon={<Globe size={14} />}
					className={`public-dashboard-btn ${
						user?.role !== USER_ROLES.ADMIN ? 'disabled-btn' : ''
					}`}
				>
					{t('publish_tab')}
				</Button>
			</Tooltip>
		),
		key: 'public-dashboard',
		children: <PublicDashboardSetting />,
		disabled: user?.role !== USER_ROLES.ADMIN,
	};

	const items = [
		{
			label: (
				<Button type="text" icon={<Table size="14" />} className="overview-btn">
					{t('overview_tab')}
				</Button>
			),
			key: 'general',
			children: <GeneralDashboardSettings />,
		},
		{
			label: (
				<Button type="text" icon={<Braces size={14} />} className="variables-btn">
					{t('variables_tab')}
				</Button>
			),
			key: 'variables',
			children: (
				<DashboardVariableSettings
					variablesSettingsTabHandle={variablesSettingsTabHandle}
				/>
			),
		},
		...(enablePublicDashboard ? [publicDashboardItem] : []),
	];

	return <Tabs items={items} animated className="settings-tabs" />;
}

export default DashboardSettings;
