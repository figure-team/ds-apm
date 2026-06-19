import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { previewNotificationTemplate } from 'api/v2/rules/previewNotificationTemplate';
import * as createAlertContext from 'container/CreateAlertV2/context';
import { createMockAlertContextState } from 'container/CreateAlertV2/EvaluationSettings/__tests__/testUtils';

import NotificationMessage from '../NotificationMessage';

jest.mock('api/v2/rules/previewNotificationTemplate', () => ({
	previewNotificationTemplate: jest.fn(),
}));

jest.mock('uplot', () => {
	const paths = {
		spline: jest.fn(),
		bars: jest.fn(),
	};
	const uplotMock = jest.fn(() => ({
		paths,
	}));
	return {
		paths,
		default: uplotMock,
	};
});

const mockSetNotificationSettings = jest.fn();
const mockPreviewNotificationTemplate =
	previewNotificationTemplate as jest.MockedFunction<
		typeof previewNotificationTemplate
	>;
const initialNotificationSettingsState =
	createMockAlertContextState().notificationSettings;
jest.spyOn(createAlertContext, 'useCreateAlertState').mockReturnValue(
	createMockAlertContextState({
		notificationSettings: {
			...initialNotificationSettingsState,
			description: '',
		},
		setNotificationSettings: mockSetNotificationSettings,
	}),
);

describe('NotificationMessage', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockPreviewNotificationTemplate.mockResolvedValue({
			status: 'success',
			data: {
				body: 'Rendered notification preview',
				missingVars: [],
			},
		});
		jest.spyOn(createAlertContext, 'useCreateAlertState').mockReturnValue(
			createMockAlertContextState({
				notificationSettings: {
					...initialNotificationSettingsState,
					description: '',
				},
				setNotificationSettings: mockSetNotificationSettings,
			}),
		);
	});

	it('renders textarea with message and placeholder', () => {
		render(<NotificationMessage />);
		expect(screen.getByText('v2_notification_message_title')).toBeInTheDocument();
		expect(screen.getByText('$incident.impact_summary')).toBeInTheDocument();
		expect(screen.getByText('$incident.next_action')).toBeInTheDocument();
		expect(screen.getByText('$incident.service_name')).toBeInTheDocument();
		expect(screen.getByText('$incident.sop_id')).toBeInTheDocument();
		const textarea = screen.getByPlaceholderText('v2_notification_message_placeholder');
		expect(textarea).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /v2_preview_btn/i })).toBeDisabled();
	});

	it('shows incident template variables in the variables popover', async () => {
		const user = userEvent.setup();
		render(<NotificationMessage />);

		await user.click(screen.getByText('v2_variables_btn'));

		await expect(
			screen.findByText('v2_incident_template_vars_title'),
		).resolves.toBeInTheDocument();
		expect(screen.getByText('$incident.vendor_request')).toBeInTheDocument();
		expect(screen.getByText('$incident.sop_url')).toBeInTheDocument();
		expect(screen.getByText('$incident.sop_source')).toBeInTheDocument();
		expect(
			screen.getByText(
				'Question or evidence request for partner/vendor developers.',
			),
		).toBeInTheDocument();
		expect(
			screen.getByText('SOP preview/deep-link URL for responders.'),
		).toBeInTheDocument();
		expect(
			screen.getByText(
				'SOP source such as Confluence, Git, Notion, or manual metadata.',
			),
		).toBeInTheDocument();
	});

	it('warns when notification message uses unknown incident variables', () => {
		jest.spyOn(createAlertContext, 'useCreateAlertState').mockReturnValue(
			createMockAlertContextState({
				notificationSettings: {
					...initialNotificationSettingsState,
					description: 'Check $incident.bad_field before $incident.next_action',
				},
				setNotificationSettings: mockSetNotificationSettings,
			}),
		);

		render(<NotificationMessage />);

		expect(screen.getByRole('alert')).toHaveTextContent(
			'v2_unknown_incident_var',
		);
	});

	it('previews notification message with alert labels and annotations', async () => {
		const user = userEvent.setup();
		jest.spyOn(createAlertContext, 'useCreateAlertState').mockReturnValue(
			createMockAlertContextState({
				alertState: {
					...createMockAlertContextState().alertState,
					annotations: {
						impact_summary: 'Checkout latency can affect payments.',
						sop_source: 'confluence',
						sop_url: 'https://runbooks.example.com/payment-latency',
					},
					labels: {
						environment: 'prod',
						'service.name': 'checkout-api',
						sop_id: 'SOP-PAY-001',
					},
				},
				notificationSettings: {
					...initialNotificationSettingsState,
					description:
						'Impact: $incident.impact_summary SOP: $incident.sop_id <$incident.sop_url> Source: $incident.sop_source',
				},
				setNotificationSettings: mockSetNotificationSettings,
			}),
		);

		render(<NotificationMessage />);

		await user.click(screen.getByRole('button', { name: /v2_preview_btn/i }));

		expect(mockPreviewNotificationTemplate).toHaveBeenCalledWith({
			template:
				'Impact: $incident.impact_summary SOP: $incident.sop_id <$incident.sop_url> Source: $incident.sop_source',
			annotations: {
				impact_summary: 'Checkout latency can affect payments.',
				sop_source: 'confluence',
				sop_url: 'https://runbooks.example.com/payment-latency',
			},
			labels: {
				environment: 'prod',
				'service.name': 'checkout-api',
				sop_id: 'SOP-PAY-001',
			},
		});
		await expect(
			screen.findByText('Rendered notification preview'),
		).resolves.toBeInTheDocument();
	});

	it('updates notification settings when textarea value changes', async () => {
		const user = userEvent.setup();
		render(<NotificationMessage />);
		const textarea = screen.getByPlaceholderText('v2_notification_message_placeholder');
		await user.type(textarea, 'x');
		expect(mockSetNotificationSettings).toHaveBeenLastCalledWith({
			type: 'SET_DESCRIPTION',
			payload: 'x',
		});
	});

	it('displays existing description value', () => {
		jest.spyOn(createAlertContext, 'useCreateAlertState').mockReturnValue(
			createMockAlertContextState({
				notificationSettings: {
					...initialNotificationSettingsState,
					description: 'Existing message',
				},
				setNotificationSettings: mockSetNotificationSettings,
			}),
		);

		render(<NotificationMessage />);

		const textarea = screen.getByDisplayValue('Existing message');
		expect(textarea).toBeInTheDocument();
	});
});
