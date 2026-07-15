import { render, screen } from 'tests/test-utils';
import { IntegrationConnectionStatus } from 'types/api/integrations/types';

import IntegrationDetailHeader from '../IntegrationDetailHeader';
import { ConnectionStates } from '../TestConnection';

jest.mock('api/Integrations/installIntegration', () => ({
	__esModule: true,
	default: jest.fn(),
}));

const baseProps = {
	id: 'redis',
	title: 'Redis',
	description: 'desc',
	icon: '',
	onUnInstallSuccess: jest.fn(),
	connectionState: ConnectionStates.NotInstalled,
	connectionData: ({
		logs: null,
		metrics: null,
	} as unknown) as IntegrationConnectionStatus,
	setActiveDetailTab: jest.fn(),
	isLoading: false,
};

describe('IntegrationDetailHeader', () => {
	it('hides the Connect button for viewers when not installed', () => {
		render(<IntegrationDetailHeader {...baseProps} />, undefined, {
			role: 'VIEWER',
		});

		expect(screen.queryByText('Connect Redis')).toBeNull();
	});

	it('shows the Connect button for editors when not installed', () => {
		render(<IntegrationDetailHeader {...baseProps} />, undefined, {
			role: 'EDITOR',
		});

		expect(screen.getByText('Connect Redis')).toBeInTheDocument();
	});

	it('keeps the Test Connection button for viewers when installed', () => {
		render(
			<IntegrationDetailHeader
				{...baseProps}
				connectionState={ConnectionStates.Connected}
			/>,
			undefined,
			{ role: 'VIEWER' },
		);

		expect(screen.getByText('Test Connection')).toBeInTheDocument();
	});
});
