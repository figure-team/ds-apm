import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import AIModuleSettings, { AIConfig, AIConfigTestResult } from './AIModuleSettings';

const mockGetAIConfig = jest.fn();
const mockTestAIConfig = jest.fn();
const mockUpdateAIConfig = jest.fn();
const mockToastSuccess = jest.fn();
const mockToastError = jest.fn();
const mockLogEvent = jest.fn();

jest.mock('api/aiModule/getAIConfig', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockGetAIConfig(...args),
}));
jest.mock('api/aiModule/testAIConfig', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockTestAIConfig(...args),
}));
jest.mock('api/aiModule/updateAIConfig', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockUpdateAIConfig(...args),
}));
jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockLogEvent(...args),
}));
jest.mock('@signozhq/ui', () => ({
	...jest.requireActual('@signozhq/ui'),
	toast: {
		success: (...args: unknown[]): unknown => mockToastSuccess(...args),
		error: (...args: unknown[]): unknown => mockToastError(...args),
	},
}));

const emptyCfg: AIConfig = {
	contractVersion: 'ds.ai_config.v1',
	orgId: 'org-1',
	provider: 'local',
	llmProvider: 'claude',
	transport: 'api',
	model: '',
	apiKey: '',
	oauthToken: '',
	binaryPath: '',
	timeoutSeconds: 0,
	updatedAt: '',
};

function resolveConfig(overrides: Partial<AIConfig> = {}): void {
	mockGetAIConfig.mockResolvedValue({ data: { ...emptyCfg, ...overrides } });
}

beforeEach(() => {
	jest.clearAllMocks();
});

describe('AIModuleSettings', () => {
	it('renders Provider radio options and waits for initial load', async () => {
		resolveConfig();
		render(<AIModuleSettings />);
		// Provider radios render after the loading spinner resolves.
		expect(await screen.findByLabelText(/Mock/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/Local/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/LLM/i)).toBeInTheDocument();
	});

	it('shows OAuth Token as multi-line TextArea when transport=CLI + llmProvider=codex', async () => {
		resolveConfig({
			provider: 'llm',
			llmProvider: 'codex',
			transport: 'cli',
		});
		render(<AIModuleSettings />);
		// Wait for initial load + form reset.
		const label = await screen.findByText('OAuth Token');
		expect(label).toBeInTheDocument();
		// antd's Input.TextArea renders a real <textarea> element; Input.Password
		// renders <input type="password">. Use placeholder to locate the field
		// then assert its tag.
		const field = screen.getByPlaceholderText(
			/Leave blank to keep the existing token/i,
		);
		expect(field.tagName).toBe('TEXTAREA');
		// Codex-specific hint copy is rendered.
		expect(
			screen.getByText(/paste an OPENAI_API_KEY/i),
		).toBeInTheDocument();
		expect(
			screen.getByText(/full `~\/\.codex\/auth\.json` JSON/i),
		).toBeInTheDocument();
	});

	it('shows OAuth Token as masked Password input when transport=CLI + llmProvider=claude', async () => {
		resolveConfig({
			provider: 'llm',
			llmProvider: 'claude',
			transport: 'cli',
		});
		render(<AIModuleSettings />);
		const label = await screen.findByText('OAuth Token');
		expect(label).toBeInTheDocument();
		const field = screen.getByPlaceholderText(
			/Leave blank to keep the existing token/i,
		);
		expect(field.tagName).toBe('INPUT');
		expect(field.getAttribute('type')).toBe('password');
		// Claude-specific hint copy.
		expect(screen.getByText(/claude setup-token/i)).toBeInTheDocument();
	});

	it('surfaces the auth banner when Test response sets errorKind="auth"', async () => {
		resolveConfig({
			provider: 'llm',
			llmProvider: 'claude',
			transport: 'cli',
		});
		const testResult: AIConfigTestResult = {
			ok: false,
			error: 'stderr: Authentication error: invalid token',
			errorKind: 'auth',
		};
		mockTestAIConfig.mockResolvedValue({ data: testResult });

		render(<AIModuleSettings />);
		// Wait for the form to be ready.
		await screen.findByText('OAuth Token');

		// Trigger Test.
		fireEvent.click(screen.getByRole('button', { name: /^Test$/i }));

		// Banner appears.
		await waitFor(() => {
			expect(
				screen.getByText(/Authentication issue detected/i),
			).toBeInTheDocument();
		});
		// Copy reflects CLI transport.
		expect(
			screen.getByText(/configured OAuth token was rejected/i),
		).toBeInTheDocument();
	});

	it('does not surface the auth banner when errorKind is not "auth"', async () => {
		resolveConfig({
			provider: 'llm',
			llmProvider: 'claude',
			transport: 'cli',
		});
		mockTestAIConfig.mockResolvedValue({
			data: { ok: false, error: 'context deadline exceeded', errorKind: 'timeout' },
		});

		render(<AIModuleSettings />);
		await screen.findByText('OAuth Token');
		fireEvent.click(screen.getByRole('button', { name: /^Test$/i }));

		// Wait for the error toast (proxy for test request finishing).
		await waitFor(() => {
			expect(mockToastError).toHaveBeenCalled();
		});
		expect(
			screen.queryByText(/Authentication issue detected/i),
		).not.toBeInTheDocument();
	});
});
