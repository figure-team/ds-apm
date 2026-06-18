import { useMemo } from 'react';
import ROUTES from 'constants/routes';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import { settingsNavSections } from './menuItems';
import { SettingsNavSection, SidebarItem } from './sideNav.types';

export type GatedSettingsSection = SettingsNavSection;

export function useSettingsNavItems(): GatedSettingsSection[] {
	const { user, trialInfo, isFetchingActiveLicense } = useAppContext();
	const {
		isCloudUser,
		isEnterpriseSelfHostedUser,
		isCommunityEnterpriseUser,
	} = useGetTenantLicense();

	const isAdmin = user.role === USER_ROLES.ADMIN;
	const isEditor = user.role === USER_ROLES.EDITOR;

	return useMemo(() => {
		const gate = (item: SidebarItem): SidebarItem => {
			let { isEnabled } = item;

			if (trialInfo?.workSpaceBlock && !isFetchingActiveLicense) {
				isEnabled = !!(
					isAdmin &&
					(item.key === ROUTES.BILLING ||
						item.key === ROUTES.ORG_SETTINGS ||
						item.key === ROUTES.MEMBERS_SETTINGS ||
						item.key === ROUTES.MY_SETTINGS ||
						item.key === ROUTES.SHORTCUTS)
				);
				return { ...item, isEnabled };
			}

			if (isCloudUser) {
				if (isAdmin) {
					isEnabled =
						item.key === ROUTES.BILLING ||
						item.key === ROUTES.ROLES_SETTINGS ||
						item.key === ROUTES.ROLE_DETAILS ||
						item.key === ROUTES.INTEGRATIONS ||
						item.key === ROUTES.INGESTION_SETTINGS ||
						item.key === ROUTES.ORG_SETTINGS ||
						item.key === ROUTES.MEMBERS_SETTINGS ||
						item.key === ROUTES.SERVICE_ACCOUNTS_SETTINGS ||
						item.key === ROUTES.SHORTCUTS ||
						item.key === ROUTES.MCP_SERVER ||
						item.key === ROUTES.AI_MODULE_SETTINGS ||
						item.key === ROUTES.CODE_RCA_SETTINGS ||
						item.key === ROUTES.INCIDENT_REPORT_SETTINGS
							? true
							: isEnabled;
				}
				if (isEditor) {
					isEnabled =
						item.key === ROUTES.INGESTION_SETTINGS ||
						item.key === ROUTES.INTEGRATIONS ||
						item.key === ROUTES.SHORTCUTS ||
						item.key === ROUTES.MCP_SERVER ||
						item.key === ROUTES.AI_MODULE_SETTINGS ||
						item.key === ROUTES.CODE_RCA_SETTINGS ||
						item.key === ROUTES.INCIDENT_REPORT_SETTINGS
							? true
							: isEnabled;
				}
			}

			if (isEnterpriseSelfHostedUser) {
				if (isAdmin) {
					isEnabled =
						item.key === ROUTES.BILLING ||
						item.key === ROUTES.ROLES_SETTINGS ||
						item.key === ROUTES.ROLE_DETAILS ||
						item.key === ROUTES.INTEGRATIONS ||
						item.key === ROUTES.ORG_SETTINGS ||
						item.key === ROUTES.MEMBERS_SETTINGS ||
						item.key === ROUTES.SERVICE_ACCOUNTS_SETTINGS ||
						item.key === ROUTES.INGESTION_SETTINGS ||
						item.key === ROUTES.MCP_SERVER ||
						item.key === ROUTES.AI_MODULE_SETTINGS ||
						item.key === ROUTES.CODE_RCA_SETTINGS ||
						item.key === ROUTES.INCIDENT_REPORT_SETTINGS
							? true
							: isEnabled;
				}
				if (isEditor) {
					isEnabled =
						item.key === ROUTES.INTEGRATIONS ||
						item.key === ROUTES.INGESTION_SETTINGS ||
						item.key === ROUTES.MCP_SERVER ||
						item.key === ROUTES.AI_MODULE_SETTINGS ||
						item.key === ROUTES.CODE_RCA_SETTINGS ||
						item.key === ROUTES.INCIDENT_REPORT_SETTINGS
							? true
							: isEnabled;
				}
			}

			if (!isCloudUser && !isEnterpriseSelfHostedUser) {
				if (isAdmin) {
					isEnabled =
						item.key === ROUTES.ORG_SETTINGS ||
						item.key === ROUTES.MEMBERS_SETTINGS ||
						item.key === ROUTES.SERVICE_ACCOUNTS_SETTINGS ||
						item.key === ROUTES.ROLES_SETTINGS ||
						item.key === ROUTES.ROLE_DETAILS ||
						item.key === ROUTES.AI_MODULE_SETTINGS ||
						item.key === ROUTES.CODE_RCA_SETTINGS ||
						item.key === ROUTES.INCIDENT_REPORT_SETTINGS
							? true
							: isEnabled;
				}
				if (item.key === ROUTES.BILLING || item.key === ROUTES.INTEGRATIONS) {
					isEnabled = false;
				}
			}

			if (item.key === ROUTES.LIST_LICENSES) {
				isEnabled = isEnterpriseSelfHostedUser || isCommunityEnterpriseUser;
			}

			return { ...item, isEnabled };
		};

		return settingsNavSections.map((section) => ({
			...section,
			items: section.items.map(gate),
		}));
	}, [
		isAdmin,
		isEditor,
		isCloudUser,
		isEnterpriseSelfHostedUser,
		isCommunityEnterpriseUser,
		isFetchingActiveLicense,
		trialInfo?.workSpaceBlock,
	]);
}
