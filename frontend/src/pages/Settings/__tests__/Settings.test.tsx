import React from 'react';
import SettingsPage from 'pages/Settings/Settings';
import { render, screen } from 'tests/test-utils';
import { LicensePlatform } from 'types/api/licensesV3/getActive';
import { USER_ROLES } from 'types/roles';

jest.mock('components/MarkdownRenderer/MarkdownRenderer', () => ({
	__esModule: true,
	default: ({ children }: { children: React.ReactNode }): React.ReactNode =>
		children,
}));

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: jest.fn(),
}));

jest.mock('lib/history', () => ({
	push: jest.fn(),
	listen: jest.fn(() => jest.fn()),
	location: { pathname: '/settings', search: '' },
}));

const getCloudAdminOverrides = (): any => ({
	activeLicense: {
		key: 'test-key',
		platform: LicensePlatform.CLOUD,
	},
});

const getSelfHostedAdminOverrides = (): any => ({
	activeLicense: {
		key: 'test-key',
		platform: LicensePlatform.SELF_HOSTED,
	},
});

describe('SettingsPage', () => {
	it.each([
		['cloud admin', USER_ROLES.ADMIN, getCloudAdminOverrides()],
		['self-hosted admin', USER_ROLES.ADMIN, getSelfHostedAdminOverrides()],
		['cloud viewer', USER_ROLES.VIEWER, getCloudAdminOverrides()],
	])(
		'renders the settings page header for %s',
		(_label, role, appContextOverrides) => {
			render(<SettingsPage />, undefined, {
				role,
				appContextOverrides,
				initialRoute: '/settings',
			});

			expect(screen.getByTestId('settings-page-title')).toBeInTheDocument();
		},
	);

	it('does not render a secondary sidenav', () => {
		render(<SettingsPage />, undefined, {
			role: USER_ROLES.ADMIN,
			appContextOverrides: getCloudAdminOverrides(),
			initialRoute: '/settings',
		});

		expect(
			screen.queryByTestId('settings-page-sidenav'),
		).not.toBeInTheDocument();
	});
});
