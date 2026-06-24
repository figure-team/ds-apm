import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { getSopDocument, listSopDocuments } from 'api/v2/rules/sopDocuments';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { defaultPostableAlertRuleV2 } from 'container/CreateAlertV2/constants';
import { getCreateAlertLocalStateFromAlertDef } from 'container/CreateAlertV2/utils';
import * as useSafeNavigateHook from 'hooks/useSafeNavigate';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import * as rulesHook from '../../../../api/generated/services/rules';
import { CreateAlertProvider } from '../../context';
import CreateAlertHeader from '../CreateAlertHeader';

jest.mock('api/v2/rules/sopDocuments', () => ({
	...jest.requireActual('api/v2/rules/sopDocuments'),
	listSopDocuments: jest.fn(),
	getSopDocument: jest.fn(),
}));

jest.mock('components/MarkdownRenderer/MarkdownRenderer', () => ({
	MarkdownRenderer: ({
		markdownContent,
	}: {
		markdownContent: string;
	}): JSX.Element => <div data-testid="markdown">{markdownContent}</div>,
}));

const mockSafeNavigate = jest.fn();
jest.spyOn(useSafeNavigateHook, 'useSafeNavigate').mockReturnValue({
	safeNavigate: mockSafeNavigate,
});

jest.spyOn(rulesHook, 'useCreateRule').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useCreateRule>);
jest.spyOn(rulesHook, 'useTestRule').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useTestRule>);
jest.spyOn(rulesHook, 'useUpdateRuleByID').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useUpdateRuleByID>);

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

jest.mock('react-router-dom', () => ({
	...jest.requireActual('react-router-dom'),
	useLocation: (): { search: string } => ({
		search: '',
	}),
}));

const ENTER_ALERT_RULE_NAME_PLACEHOLDER = 'v2_alert_name_placeholder';
const mockListSopDocuments = listSopDocuments as jest.MockedFunction<
	typeof listSopDocuments
>;
const mockGetSopDocument = getSopDocument as jest.MockedFunction<
	typeof getSopDocument
>;

const SOP_DOC_SUMMARY = {
	contractVersion: 'ds.sop_document.v1',
	sopId: 'SOP-PAY-001',
	title: 'Payment API 5xx response',
	version: '2026-04-20.3',
	checksum: 'sha256:abc',
	source: { type: 'managed_markdown', sourceId: 'confluence' },
	displayUrl: 'kb.example/sop/SOP-PAY-001',
	ownerTeam: 'payments-team',
	approvalStatus: 'approved',
	tenantScope: { projectIds: [], environments: [] },
	updatedAt: '2026-04-20T00:00:00Z',
};

const renderCreateAlertHeader = (): ReturnType<typeof render> =>
	render(
		<CreateAlertProvider initialAlertType={AlertTypes.METRICS_BASED_ALERT}>
			<CreateAlertHeader />
		</CreateAlertProvider>,
	);

describe('CreateAlertHeader', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockListSopDocuments.mockResolvedValue({
			status: 'success',
			data: {
				contractVersion: 'ds.sop_document_list.v1',
				documents: [SOP_DOC_SUMMARY],
			},
		} as never);
		mockGetSopDocument.mockResolvedValue({
			status: 'success',
			data: {
				...SOP_DOC_SUMMARY,
				bodyMarkdown: '## Payment API 5xx 대응\n1. 대시보드 확인',
				customerUpdateTemplate: '현재 결제 일부에서 지연이 발생하고 있습니다.',
				vendorRequestTemplate: '',
				securityContext: {
					serviceAccountProfile: 'ds-sop-reader',
					secretRefVisible: false,
					browserCredentialsUsed: false,
					redactionApplied: false,
				},
			},
		} as never);
	});

	it('renders the header with title', () => {
		renderCreateAlertHeader();
		expect(screen.getByText('v2_new_alert_rule')).toBeInTheDocument();
	});

	it('renders name input with placeholder', () => {
		renderCreateAlertHeader();
		const nameInput = screen.getByPlaceholderText(
			ENTER_ALERT_RULE_NAME_PLACEHOLDER,
		);
		expect(nameInput).toBeInTheDocument();
	});

	it('renders LabelsInput component', () => {
		renderCreateAlertHeader();
		expect(screen.getByText('v2_add_labels_btn')).toBeInTheDocument();
	});

	it('shows missing SI/SM routing metadata labels for new alerts', () => {
		renderCreateAlertHeader();

		expect(screen.getByText('v2_sisam_routing_title')).toBeInTheDocument();
		expect(screen.getByText('v2_missing_labels')).toBeInTheDocument();
		expect(screen.getByText('service.name')).toBeInTheDocument();
		expect(screen.getByText('owner_team')).toBeInTheDocument();
	});

	it('shows complete SI/SM routing metadata when required labels are present', () => {
		render(
			<CreateAlertProvider
				isEditMode
				initialAlertType={AlertTypes.METRICS_BASED_ALERT}
				initialAlertState={getCreateAlertLocalStateFromAlertDef({
					...defaultPostableAlertRuleV2,
					labels: {
						environment: 'prod',
						owner_team: 'sm-payments',
						project_id: 'customer-a',
						'service.name': 'payment-api',
					},
				})}
			>
				<CreateAlertHeader />
			</CreateAlertProvider>,
		);

		expect(
			screen.getByText('v2_all_labels_present'),
		).toBeInTheDocument();
		expect(screen.getAllByText('v2_label_set')).toHaveLength(4);
	});

	it('renders SOP binding metadata and updates label plus annotations', () => {
		renderCreateAlertHeader();

		expect(screen.getByText('v2_sop_binding_title')).toBeInTheDocument();
		expect(screen.getByText('v2_sop_binding_missing')).toBeInTheDocument();

		const sopIdInput = screen.getByTestId('sop-metadata-sop_id');
		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');

		fireEvent.change(sopIdInput, {
			target: { value: 'SOP-PAY-001' },
		});
		fireEvent.change(sopUrlInput, {
			target: { value: 'https://kb.example/sop/SOP-PAY-001' },
		});

		expect(sopIdInput).toHaveValue('SOP-PAY-001');
		expect(sopUrlInput).toHaveValue('https://kb.example/sop/SOP-PAY-001');
		expect(screen.getByText('v2_sop_binding_present')).toBeInTheDocument();
	});

	it('toggles the SOP document/template preview and renders sections', async () => {
		renderCreateAlertHeader();

		// Wait for SOP documents to load so the bound document can resolve.
		await waitFor(() => expect(mockListSopDocuments).toHaveBeenCalled());

		fireEvent.change(screen.getByTestId('sop-metadata-sop_id'), {
			target: { value: 'SOP-PAY-001' },
		});

		fireEvent.click(
			screen.getByRole('button', { name: 'v2_sop_preview_expand' }),
		);

		await screen.findByTestId('sop-doc-preview');

		await waitFor(() =>
			expect(mockGetSopDocument).toHaveBeenCalledWith(
				'SOP-PAY-001',
				'2026-04-20.3',
			),
		);

		// Body markdown and the customer template render; the empty vendor
		// template falls back to the muted placeholder.
		expect(await screen.findByText(/Payment API 5xx/)).toBeInTheDocument();
		expect(
			screen.getByText('현재 결제 일부에서 지연이 발생하고 있습니다.'),
		).toBeInTheDocument();
		expect(
			screen.getByText('v2_sop_customer_template_section'),
		).toBeInTheDocument();
		expect(
			screen.getByText('v2_sop_vendor_template_section'),
		).toBeInTheDocument();
		expect(screen.getByText('v2_sop_doc_empty')).toBeInTheDocument();

		// Collapsing hides the preview body again.
		fireEvent.click(
			screen.getByRole('button', { name: 'v2_sop_preview_collapse' }),
		);
		expect(screen.queryByTestId('sop-doc-preview')).not.toBeInTheDocument();
	});

	it('warns when SOP metadata uses unsafe values', () => {
		renderCreateAlertHeader();

		const sopIdInput = screen.getByTestId('sop-metadata-sop_id');
		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');

		fireEvent.change(sopIdInput, {
			target: { value: `SOP-${'a'.repeat(121)}` },
		});
		fireEvent.change(sopUrlInput, {
			target: { value: 'javascript:alert(1)' },
		});

		expect(
			screen.getByText('Keep sop_id under 120 characters.'),
		).toBeInTheDocument();
		expect(
			screen.getByText('Use an http:// or https:// SOP URL.'),
		).toBeInTheDocument();
		expect(sopIdInput).toHaveAttribute('aria-invalid', 'true');
		expect(sopUrlInput).toHaveAttribute('aria-invalid', 'true');
	});

	it('warns when SOP URL includes browser-visible credentials', () => {
		renderCreateAlertHeader();

		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');

		fireEvent.change(sopUrlInput, {
			target: { value: 'https://user:pass@kb.example/sop/SOP-PAY-001' },
		});

		expect(
			screen.getByText(
				'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
			),
		).toBeInTheDocument();
		expect(sopUrlInput).toHaveAttribute('aria-invalid', 'true');
	});

	it('updates name when typing in name input', () => {
		renderCreateAlertHeader();
		const nameInput = screen.getByPlaceholderText(
			ENTER_ALERT_RULE_NAME_PLACEHOLDER,
		);

		fireEvent.change(nameInput, { target: { value: 'Test Alert' } });

		expect(nameInput).toHaveValue('Test Alert');
	});

	it('renders the header with title when isEditMode is true', () => {
		render(
			<CreateAlertProvider
				isEditMode
				initialAlertType={AlertTypes.METRICS_BASED_ALERT}
				initialAlertState={getCreateAlertLocalStateFromAlertDef(
					defaultPostableAlertRuleV2,
				)}
			>
				<CreateAlertHeader />
			</CreateAlertProvider>,
		);
		expect(screen.queryByText('v2_new_alert_rule')).not.toBeInTheDocument();
		expect(
			screen.getByPlaceholderText(ENTER_ALERT_RULE_NAME_PLACEHOLDER),
		).toHaveValue('TEST_ALERT');
	});

	it('should navigate to classic experience when button is clicked', () => {
		renderCreateAlertHeader();
		const switchToClassicExperienceButton = screen.getByText(
			'v2_switch_to_classic',
		);
		expect(switchToClassicExperienceButton).toBeInTheDocument();
		fireEvent.click(switchToClassicExperienceButton);

		const params = new URLSearchParams();
		params.set(QueryParams.showClassicCreateAlertsPage, 'true');
		expect(mockSafeNavigate).toHaveBeenCalledWith(
			`${ROUTES.ALERTS_NEW}?${params.toString()}`,
			{ replace: true },
		);
	});

	it('should not render "switch to classic experience" button when isEditMode is true', () => {
		render(
			<CreateAlertProvider
				isEditMode
				initialAlertType={AlertTypes.METRICS_BASED_ALERT}
				initialAlertState={getCreateAlertLocalStateFromAlertDef(
					defaultPostableAlertRuleV2,
				)}
			>
				<CreateAlertHeader />
			</CreateAlertProvider>,
		);
		expect(
			screen.queryByText('v2_switch_to_classic'),
		).not.toBeInTheDocument();
	});
});
