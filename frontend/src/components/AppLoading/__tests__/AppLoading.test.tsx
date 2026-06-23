import { render, screen } from '@testing-library/react';

import getLocal from '../../../api/browser/localstorage/get';
import AppLoading from '../AppLoading';

jest.mock('../../../api/browser/localstorage/get', () => ({
	__esModule: true,
	default: jest.fn(),
}));

// Access the mocked function
const mockGet = getLocal as unknown as jest.Mock;

describe('AppLoading', () => {
	const SIGNOZ_TEXT = 'DS-APM';
	const LOGO_ALT = 'kt ds';
	const CONTAINER_SELECTOR = '.app-loading-container';

	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('should render loading screen with dark theme by default', () => {
		// Mock localStorage to return dark theme (or undefined for default)
		mockGet.mockReturnValue(undefined);

		render(<AppLoading />);

		// Check if main elements are rendered
		expect(screen.getByAltText(LOGO_ALT)).toBeInTheDocument();
		expect(screen.getByText(SIGNOZ_TEXT)).toBeInTheDocument();

		// Check if dark theme class is applied
		const container = screen.getByText(SIGNOZ_TEXT).closest(CONTAINER_SELECTOR);
		expect(container).toHaveClass('dark');
		expect(container).not.toHaveClass('lightMode');
	});

	it('should have proper structure and content', () => {
		// Mock localStorage to return dark theme
		mockGet.mockReturnValue(undefined);

		render(<AppLoading />);

		// Check for brand logo
		const logo = screen.getByAltText(LOGO_ALT);
		expect(logo).toBeInTheDocument();
		expect(logo).toHaveAttribute('src', 'test-file-stub');

		// Check for brand title
		const title = screen.getByText(SIGNOZ_TEXT);
		expect(title).toBeInTheDocument();

		// Check for loader
		const loader = document.querySelector('.loader');
		expect(loader).toBeInTheDocument();
	});

	it('should handle localStorage errors gracefully', () => {
		// Mock localStorage to throw an error
		mockGet.mockImplementation(() => {
			throw new Error('localStorage not available');
		});

		render(<AppLoading />);

		// Should still render with dark theme as fallback
		expect(screen.getByText(SIGNOZ_TEXT)).toBeInTheDocument();
		const container = screen.getByText(SIGNOZ_TEXT).closest(CONTAINER_SELECTOR);
		expect(container).toHaveClass('dark');
	});
});
