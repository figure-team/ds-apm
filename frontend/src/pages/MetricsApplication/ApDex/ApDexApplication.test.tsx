import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import { thresholdMockData } from './__mock__/thresholdMockData';
import ApDexApplication from './ApDexApplication';

jest.mock('react-i18next', () => ({
	useTranslation: (): {
		t: (str: string) => string;
		i18n: { changeLanguage: () => Promise<void> };
	} => ({
		t: (str: string): string => str,
		i18n: { changeLanguage: (): Promise<void> => Promise.resolve() },
	}),
}));

jest.mock('react-router-dom', () => ({
	...jest.requireActual('react-router-dom'),
	useParams: (): {
		servicename: string;
	} => ({ servicename: 'mockServiceName' }),
}));

jest.mock('hooks/apDex/useGetApDexSettings', () => ({
	__esModule: true,
	useGetApDexSettings: jest.fn().mockReturnValue({
		data: thresholdMockData,
		isLoading: false,
		error: null,
		refetch: jest.fn(),
	}),
}));

jest.mock('hooks/apDex/useSetApDexSettings', () => ({
	__esModule: true,
	useSetApDexSettings: jest.fn().mockReturnValue({
		mutateAsync: jest.fn(),
		isLoading: false,
		error: null,
	}),
}));

describe('ApDexApplication', () => {
	it('should render the component', () => {
		render(<ApDexApplication />);

		expect(screen.getByText('services:settings')).toBeInTheDocument();
	});

	it('should open the popover when the settings button is clicked', async () => {
		render(<ApDexApplication />);

		const button = screen.getByText('services:settings');
		fireEvent.click(button);
		await waitFor(() => {
			expect(
				screen.getByText('services:application_settings'),
			).toBeInTheDocument();
		});
	});

	it('should close the popover when the close button is clicked', async () => {
		render(<ApDexApplication />);

		const button = screen.getByText('services:settings');
		fireEvent.click(button);
		await waitFor(() => {
			expect(
				screen.getByText('services:application_settings'),
			).toBeInTheDocument();
		});

		const closeButton = screen.getByText('common:cancel');
		fireEvent.click(closeButton);
		await waitFor(() => {
			expect(
				screen.queryByText('services:application_settings'),
			).not.toBeInTheDocument();
		});
	});
});
