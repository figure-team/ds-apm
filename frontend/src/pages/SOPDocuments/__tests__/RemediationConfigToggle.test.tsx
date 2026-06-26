import { render, screen } from 'tests/test-utils';

import RemediationConfigToggle from '../RemediationConfigToggle';

const mockGet = jest.fn();
const mockUpdate = jest.fn();

jest.mock('api/remediation', () => ({
	getRemediationConfig: (...args: unknown[]): unknown => mockGet(...args),
	updateRemediationConfig: (...args: unknown[]): unknown => mockUpdate(...args),
}));

const enabledConfig = {
	executionEnabled: true,
	proposalTtlSeconds: 1800,
	execTimeoutSeconds: 300,
	verifyWindowSeconds: 600,
	maxConcurrent: 1,
};

describe('RemediationConfigToggle', () => {
	afterEach(() => {
		jest.clearAllMocks();
	});

	it('renders nothing and skips the API for non-admins', () => {
		mockGet.mockResolvedValue(enabledConfig);

		render(<RemediationConfigToggle />, undefined, { role: 'VIEWER' });

		expect(screen.queryByTestId('remediation-config-toggle')).toBeNull();
		expect(mockGet).not.toHaveBeenCalled();
	});

	it('shows the toggle and loads the org config for admins', async () => {
		mockGet.mockResolvedValue(enabledConfig);

		render(<RemediationConfigToggle />, undefined, { role: 'ADMIN' });

		// Title renders immediately; status appears once the config resolves.
		expect(screen.getByText('remediation_toggle_title')).toBeInTheDocument();
		expect(
			await screen.findByText('remediation_toggle_status_enabled'),
		).toBeInTheDocument();
		// Called at least once (StrictMode may double-invoke the effect in tests).
		expect(mockGet).toHaveBeenCalled();
	});
});
