import React from 'react';
import { fireEvent, screen } from '@testing-library/react';
import { render } from 'tests/test-utils';
import ROUTES from 'constants/routes';

// jest.mock is hoisted before any variable initialisation.
// Use inline factory + jest.requireMock to retrieve the spy.
jest.mock('lib/history', () => ({
	push: jest.fn(),
	listen: jest.fn(() => jest.fn()),
	location: { pathname: '/', search: '' },
}));

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: jest.fn(),
}));

jest.mock('hooks/hotkeys/useKeyboardHotkeys', () => ({
	useKeyboardHotkeys: (): any => ({
		registerShortcut: jest.fn(),
		deregisterShortcut: jest.fn(),
	}),
}));

jest.mock('hooks/useComponentPermission', () => ({
	__esModule: true,
	default: (): [boolean] => [true],
}));

jest.mock('providers/cmdKProvider', () => ({
	useCmdK: (): any => ({ openCmdK: jest.fn() }),
}));

// eslint-disable-next-line import/first
import SideNav from '../SideNav';

describe('SideNav settings group', () => {
	// Access the mock via requireMock after the module has been mocked
	// eslint-disable-next-line @typescript-eslint/no-var-requires
	const historyMock = jest.requireMock('lib/history');

	beforeEach(() => {
		(historyMock.push as jest.Mock).mockClear();
	});

	it('settings group header navigates to MY_SETTINGS and expands to show account item', () => {
		// Render SideNav pinned so the sidebar is not collapsed (isPinned=true keeps it open)
		render(<SideNav isPinned />, undefined, {
			role: 'ADMIN',
			initialRoute: '/',
		});

		// The settings group header must be present
		const header = screen.getByTestId('settings-group-header');
		expect(header).toBeInTheDocument();

		// Click the header
		fireEvent.click(header);

		// Should have navigated to MY_SETTINGS
		expect(historyMock.push).toHaveBeenCalledWith(ROUTES.MY_SETTINGS);

		// After expanding, the account nav item should be visible (i18n mock returns key)
		expect(screen.getByText('routes:account')).toBeInTheDocument();
	});
});
