import { render, screen } from 'tests/test-utils';

import IntergrationsUninstallBar from '../IntegrationsUninstallBar';
import { ConnectionStates } from '../TestConnection';

jest.mock('api/Integrations/uninstallIntegration', () => ({
	__esModule: true,
	default: jest.fn(),
}));

describe('IntegrationsUninstallBar', () => {
	it('hides the remove bar for viewers', () => {
		render(
			<IntergrationsUninstallBar
				integrationTitle="Redis"
				integrationId="redis"
				onUnInstallSuccess={jest.fn()}
				connectionStatus={ConnectionStates.Connected}
			/>,
			undefined,
			{ role: 'VIEWER' },
		);

		// i18n mock은 키를 반환한다.
		expect(screen.queryByText('uninstall.remove_integration')).toBeNull();
	});

	it('shows the remove bar for editors', () => {
		render(
			<IntergrationsUninstallBar
				integrationTitle="Redis"
				integrationId="redis"
				onUnInstallSuccess={jest.fn()}
				connectionStatus={ConnectionStates.Connected}
			/>,
			undefined,
			{ role: 'EDITOR' },
		);

		expect(screen.getByText('uninstall.remove_integration')).toBeInTheDocument();
	});
});
