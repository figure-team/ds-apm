import { render, screen } from '@testing-library/react';
import ROUTES from 'constants/routes';
import * as useGetTenantLicense from 'hooks/useGetTenantLicense';
import * as useSafeNavigate from 'hooks/useSafeNavigate';
import { userEvent } from 'tests/test-utils';

import AlertNotFound from '../AlertNotFound';

jest.mock('lib/history', () => ({
	__esModule: true,
	default: {
		push: jest.fn(),
	},
}));

import history from 'lib/history';

const mockSafeNavigate = jest.fn();
const useGetTenantLicenseSpy = jest.spyOn(
	useGetTenantLicense,
	'useGetTenantLicense',
);
const useSafeNavigateSpy = jest.spyOn(useSafeNavigate, 'useSafeNavigate');

describe('AlertNotFound', () => {
	beforeEach(() => {
		mockSafeNavigate.mockClear();
		window.open = jest.fn();
		useGetTenantLicenseSpy.mockReturnValue({
			isCloudUser: false,
		} as ReturnType<typeof useGetTenantLicense.useGetTenantLicense>);
		useSafeNavigateSpy.mockReturnValue({
			safeNavigate: mockSafeNavigate,
		});
	});

	it('should render the correct error message for test alerts', () => {
		render(<AlertNotFound isTestAlert />);
		expect(
			screen.getByText('not_found_message'),
		).toBeInTheDocument();
		expect(
			screen.getByText('not_found_test_alert'),
		).toBeInTheDocument();
	});

	it('should render the correct error message for non-existing alerts', () => {
		render(<AlertNotFound isTestAlert={false} />);
		expect(
			screen.getByText('not_found_message'),
		).toBeInTheDocument();
		expect(
			screen.getByText('not_found_link_incorrect'),
		).toBeInTheDocument();
		expect(
			screen.getByText('not_found_deleted'),
		).toBeInTheDocument();
	});

	it('should navigate to the list all alerts page when the check all rules button is clicked', async () => {
		const user = userEvent.setup();
		render(<AlertNotFound isTestAlert={false} />);
		await user.click(screen.getByText('not_found_check_rules'));
		expect(mockSafeNavigate).toHaveBeenCalledWith(ROUTES.LIST_ALL_ALERT, {
			newTab: false,
		});
	});

	it('should navigate to the correct support page for cloud users when button is clicked', async () => {
		const user = userEvent.setup();
		useGetTenantLicenseSpy.mockReturnValueOnce({
			isCloudUser: true,
		} as ReturnType<typeof useGetTenantLicense.useGetTenantLicense>);

		render(<AlertNotFound isTestAlert={false} />);
		await user.click(screen.getByText('not_found_contact_support'));
		expect(history.push).toHaveBeenCalledWith('/support');
	});

	it('should navigate to the support page for self-hosted users when the contact support button is clicked', async () => {
		const user = userEvent.setup();
		render(<AlertNotFound isTestAlert={false} />);
		await user.click(screen.getByText('not_found_contact_support'));
		expect(window.open).toHaveBeenCalledWith('https://signoz.io/slack', '_blank');
	});
});
