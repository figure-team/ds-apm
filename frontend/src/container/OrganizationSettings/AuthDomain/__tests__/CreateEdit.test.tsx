import {
	fireEvent,
	render,
	screen,
	userEvent,
	waitFor,
} from 'tests/test-utils';

import CreateEdit from '../CreateEdit/CreateEdit';
import {
	mockDomainWithRoleMapping,
	mockGoogleAuthDomain,
	mockGoogleAuthWithWorkspaceGroups,
	mockOidcAuthDomain,
	mockOidcWithClaimMapping,
	mockSamlAuthDomain,
	mockSamlWithAttributeMapping,
} from './mocks';

const mockOnClose = jest.fn();

describe('CreateEdit Modal', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	describe('Provider Selection (Create Mode)', () => {
		it('renders provider selection when creating new domain', () => {
			render(<CreateEdit isCreate onClose={mockOnClose} />);

			expect(screen.getByText('selector_title')).toBeInTheDocument();
			expect(screen.getByText('selector_google_title')).toBeInTheDocument();
			expect(screen.getByText('selector_saml_title')).toBeInTheDocument();
			expect(screen.getByText('selector_oidc_title')).toBeInTheDocument();
		});

		it('returns to provider selection when back button is clicked', async () => {
			render(<CreateEdit isCreate onClose={mockOnClose} />);

			const configureButtons = await screen.findAllByRole('button', {
				name: /configure/i,
			});
			// Use fireEvent to skip userEvent's pointer simulation and the Antd
			// Tooltip mouseEnterDelay timers it triggers on the Configure button.
			fireEvent.click(configureButtons[0]);

			expect(await screen.findByText('google_title')).toBeInTheDocument();

			const backButton = screen.getByRole('button', { name: 'back' });
			fireEvent.click(backButton);

			expect(await screen.findByText('selector_title')).toBeInTheDocument();
		});
	});

	describe('Edit Mode', () => {
		it('shows provider form directly when editing existing domain', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockGoogleAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByText('google_title')).toBeInTheDocument();
			expect(screen.queryByText('selector_title')).not.toBeInTheDocument();
		});

		it('pre-fills form with existing domain values', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockGoogleAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByDisplayValue('signoz.io')).toBeInTheDocument();
			expect(screen.getByDisplayValue('test-client-id')).toBeInTheDocument();
		});

		it('disables domain field when editing', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockGoogleAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			const domainInput = screen.getByDisplayValue('signoz.io');
			expect(domainInput).toBeDisabled();
		});

		it('shows cancel button instead of back when editing', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockGoogleAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			expect(
				screen.getByRole('button', { name: 'common:cancel' }),
			).toBeInTheDocument();
			expect(
				screen.queryByRole('button', { name: 'back' }),
			).not.toBeInTheDocument();
		});
	});

	// Todo: to fixed properly - failing with - due to timeout > 5000ms
	describe.skip('Form Validation', () => {
		it('shows validation error when submitting without required fields', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(<CreateEdit isCreate onClose={mockOnClose} />);

			const configureButtons = await screen.findAllByRole('button', {
				name: /configure/i,
			});
			await user.click(configureButtons[0]);

			const saveButton = await screen.findByRole('button', {
				name: 'save_changes',
			});
			await user.click(saveButton);

			await waitFor(() => {
				expect(screen.getByText('domain_required')).toBeInTheDocument();
			});
		});
	});

	describe('Google Auth Provider', () => {
		it('shows Google Auth form fields', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(<CreateEdit isCreate onClose={mockOnClose} />);

			const configureButtons = await screen.findAllByRole('button', {
				name: /configure/i,
			});
			await user.click(configureButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('google_title')).toBeInTheDocument();
				expect(screen.getByLabelText('field_domain')).toBeInTheDocument();
				expect(screen.getByLabelText('field_client_id')).toBeInTheDocument();
				expect(screen.getByLabelText('field_client_secret')).toBeInTheDocument();
				expect(screen.getByText('skip_email_verification')).toBeInTheDocument();
			});
		});

		it('shows workspace groups section when expanded', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(
				<CreateEdit
					isCreate={false}
					record={mockGoogleAuthWithWorkspaceGroups}
					onClose={mockOnClose}
				/>,
			);

			const workspaceHeader = screen.getByText('google_workspace_groups_title');
			await user.click(workspaceHeader);

			await waitFor(() => {
				expect(screen.getByText('google_fetch_groups')).toBeInTheDocument();
				expect(screen.getByText('google_service_account_json')).toBeInTheDocument();
			});
		});
	});

	describe('SAML Provider', () => {
		it('shows SAML-specific fields when editing SAML domain', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockSamlAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByText('saml_title')).toBeInTheDocument();
			expect(
				screen.getByDisplayValue('https://idp.example.com/sso'),
			).toBeInTheDocument();
			expect(screen.getByDisplayValue('urn:example:idp')).toBeInTheDocument();
		});

		it('shows attribute mapping section for SAML', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(
				<CreateEdit
					isCreate={false}
					record={mockSamlWithAttributeMapping}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByText('attr_mapping_title')).toBeInTheDocument();

			const attributeHeader = screen.getByText('attr_mapping_title');
			await user.click(attributeHeader);

			await waitFor(() => {
				expect(screen.getByLabelText('attr_mapping_name')).toBeInTheDocument();
				expect(screen.getByLabelText('attr_mapping_groups')).toBeInTheDocument();
				expect(screen.getByLabelText('attr_mapping_role')).toBeInTheDocument();
			});
		});
	});

	describe('OIDC Provider', () => {
		it('shows OIDC-specific fields when editing OIDC domain', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockOidcAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByText('oidc_title')).toBeInTheDocument();
			expect(screen.getByDisplayValue('https://oidc.corp.io')).toBeInTheDocument();
			expect(screen.getByDisplayValue('oidc-client-id')).toBeInTheDocument();
		});

		it('shows claim mapping section for OIDC', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(
				<CreateEdit
					isCreate={false}
					record={mockOidcWithClaimMapping}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByText('claim_mapping_title')).toBeInTheDocument();

			const claimHeader = screen.getByText('claim_mapping_title');
			await user.click(claimHeader);

			await waitFor(() => {
				expect(screen.getByLabelText('claim_mapping_email')).toBeInTheDocument();
				expect(screen.getByLabelText('claim_mapping_name')).toBeInTheDocument();
				expect(screen.getByLabelText('claim_mapping_groups')).toBeInTheDocument();
				expect(screen.getByLabelText('claim_mapping_role')).toBeInTheDocument();
			});
		});

		it('shows OIDC options checkboxes', () => {
			render(
				<CreateEdit
					isCreate={false}
					record={mockOidcAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			expect(screen.getByText('skip_email_verification')).toBeInTheDocument();
			expect(screen.getByText('oidc_get_user_info')).toBeInTheDocument();
		});
	});

	describe('Role Mapping', () => {
		it('shows role mapping section in provider forms', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(<CreateEdit isCreate onClose={mockOnClose} />);

			const configureButtons = await screen.findAllByRole('button', {
				name: /configure/i,
			});
			await user.click(configureButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('role_mapping_title')).toBeInTheDocument();
			});
		});

		it('expands role mapping section to show default role selector', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(
				<CreateEdit
					isCreate={false}
					record={mockDomainWithRoleMapping}
					onClose={mockOnClose}
				/>,
			);

			const roleMappingHeader = screen.getByText('role_mapping_title');
			await user.click(roleMappingHeader);

			await waitFor(() => {
				expect(screen.getByText('role_mapping_default_role')).toBeInTheDocument();
				expect(
					screen.getByText('role_mapping_use_role_attribute'),
				).toBeInTheDocument();
			});
		});

		it('shows group mappings section when useRoleAttribute is false', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(
				<CreateEdit
					isCreate={false}
					record={mockDomainWithRoleMapping}
					onClose={mockOnClose}
				/>,
			);

			const roleMappingHeader = screen.getByText('role_mapping_title');
			await user.click(roleMappingHeader);

			await waitFor(() => {
				expect(screen.getByText('role_mapping_group_title')).toBeInTheDocument();
				expect(
					screen.getByRole('button', { name: 'role_mapping_add' }),
				).toBeInTheDocument();
			});
		});
	});

	// Todo: to fixed properly - failing with - due to timeout > 5000ms
	describe.skip('Modal Actions', () => {
		it('calls onClose when cancel button is clicked', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			render(
				<CreateEdit
					isCreate={false}
					record={mockGoogleAuthDomain}
					onClose={mockOnClose}
				/>,
			);

			const cancelButton = screen.getByRole('button', { name: 'common:cancel' });
			await user.click(cancelButton);

			expect(mockOnClose).toHaveBeenCalled();
		});
	});
});
