import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import CodeRcaSettings from './CodeRcaSettings';

// ── API mocks ──────────────────────────────────────────────────────────────
const mockGetConfig = jest.fn();
const mockUpdateConfig = jest.fn();
const mockListRepos = jest.fn();
const mockUpsertRepo = jest.fn();
const mockDeleteRepo = jest.fn();
const mockListServiceMaps = jest.fn();
const mockUpsertServiceMap = jest.fn();
const mockDeleteServiceMap = jest.fn();
const mockListRuns = jest.fn();
const mockGetRun = jest.fn();

const mockToastSuccess = jest.fn();
const mockToastError = jest.fn();

jest.mock('api/codeRca/getConfig', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockGetConfig(...args),
}));
jest.mock('api/codeRca/updateConfig', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockUpdateConfig(...args),
}));
jest.mock('api/codeRca/listRepos', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockListRepos(...args),
}));
jest.mock('api/codeRca/upsertRepo', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockUpsertRepo(...args),
}));
jest.mock('api/codeRca/deleteRepo', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockDeleteRepo(...args),
}));
jest.mock('api/codeRca/listServiceMaps', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockListServiceMaps(...args),
}));
jest.mock('api/codeRca/upsertServiceMap', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockUpsertServiceMap(...args),
}));
jest.mock('api/codeRca/deleteServiceMap', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockDeleteServiceMap(...args),
}));
jest.mock('api/codeRca/listRuns', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockListRuns(...args),
}));
jest.mock('api/codeRca/getRun', () => ({
	__esModule: true,
	default: (...args: unknown[]): unknown => mockGetRun(...args),
}));
jest.mock('@signozhq/ui', () => ({
	...jest.requireActual('@signozhq/ui'),
	toast: {
		success: (...args: unknown[]): unknown => mockToastSuccess(...args),
		error: (...args: unknown[]): unknown => mockToastError(...args),
	},
}));

// ── Default fixture data ───────────────────────────────────────────────────
const defaultConfig = {
	contractVersion: 'ds.coderca_config.v1',
	orgId: 'org-1',
	enabled: false,
	minSeverity: 'error' as const,
	cooldownWindowSecs: 300,
	maxRunsPerDay: 10,
	maxQueueDepth: 5,
	maxConcurrentRuns: 2,
	allowUnboundWithoutAnomaly: false,
	updatedAt: '',
};

const defaultRepo = {
	contractVersion: 'ds.codebase_repo.v1',
	orgId: 'org-1',
	repoId: 'repo-abc',
	gitUrl: 'https://github.com/org/repo',
	defaultBranch: 'main',
	credential: '<unchanged>',
	enabled: true,
	branchName: 'main',
	fetched: true,
	baselineCommit: 'abc12345',
	lastSyncAt: '',
	lastSyncStatus: 'ok',
};

const defaultRun = {
	runId: 'run-1',
	orgId: 'org-1',
	service: 'my-service',
	status: 'done' as const,
	baselineCommit: 'abc12345def67890',
	createdAt: 1700000000,
	finishedAt: 1700001000,
	attempts: 1,
	resultRef: '',
};

const defaultRunDetail = {
	...defaultRun,
	rootCause: 'NullPointerException in line 42',
	proposedFix: 'Add null check before usage',
	confidence: 'high',
	limitations: 'Only covers Java files',
};

function setupDefaultMocks(): void {
	mockGetConfig.mockResolvedValue({ data: defaultConfig });
	mockListRepos.mockResolvedValue({ data: [defaultRepo] });
	mockListServiceMaps.mockResolvedValue({ data: [] });
	mockListRuns.mockResolvedValue({ data: [defaultRun] });
	mockGetRun.mockResolvedValue({ data: defaultRunDetail });
	mockUpdateConfig.mockResolvedValue({ data: {} });
	mockUpsertRepo.mockResolvedValue({ data: {} });
	mockDeleteRepo.mockResolvedValue({ data: {} });
	mockUpsertServiceMap.mockResolvedValue({ data: {} });
	mockDeleteServiceMap.mockResolvedValue({ data: {} });
}

beforeEach(() => {
	jest.clearAllMocks();
	setupDefaultMocks();
});

describe('CodeRcaSettings', () => {
	// Case 1: initial load calls getConfig/listRepos/listServiceMaps + renders tab_config
	it('loads initial data and renders the config tab', async () => {
		render(<CodeRcaSettings />);

		// tab_config label rendered (i18n mock returns key)
		expect(await screen.findByText('tab_config')).toBeInTheDocument();
		expect(screen.getByText('tab_runs')).toBeInTheDocument();

		// verify APIs called on mount
		await waitFor(() => {
			expect(mockGetConfig).toHaveBeenCalledTimes(1);
			expect(mockListRepos).toHaveBeenCalledTimes(1);
			expect(mockListServiceMaps).toHaveBeenCalledTimes(1);
		});
	});

	// Case 2: toggle enabled + save → updateConfig called with correct payload
	it('toggles enabled switch and calls updateConfig on save', async () => {
		render(<CodeRcaSettings />);

		// wait for config tab to be rendered with save button
		const saveBtn = await screen.findByRole('button', { name: 'save' });

		// find the enabled switch - initially unchecked (enabled: false)
		const switches = screen.getAllByRole('switch');
		const enabledSwitch = switches[0];
		expect(enabledSwitch).toBeInTheDocument();

		// toggle it on
		fireEvent.click(enabledSwitch);

		// click save
		fireEvent.click(saveBtn);

		await waitFor(() => {
			expect(mockUpdateConfig).toHaveBeenCalledTimes(1);
			const payload = mockUpdateConfig.mock.calls[0][0];
			expect(payload.enabled).toBe(true);
		});

		await waitFor(() => {
			expect(mockToastSuccess).toHaveBeenCalled();
		});
	});

	// Case 3a: repo add submit → upsertRepo called with credential verbatim
	it('adds a new repo via modal and calls upsertRepo with credential verbatim', async () => {
		// fresh listRepos after upsert
		mockListRepos
			.mockResolvedValueOnce({ data: [] })
			.mockResolvedValue({ data: [defaultRepo] });

		render(<CodeRcaSettings />);

		// wait for repos card add button to appear
		const addRepoBtn = await screen.findByRole('button', { name: 'repo_add' });
		fireEvent.click(addRepoBtn);

		// modal opens
		await waitFor(() => {
			expect(screen.getByRole('dialog')).toBeInTheDocument();
		});

		// antd Form.Item renders inputs with id = form_item_name (the form name + field name)
		// Use querySelectorAll to get all inputs inside the modal dialog
		const dialog = screen.getByRole('dialog');
		const allInputs = dialog.querySelectorAll('input');

		// Order in the form: repoId (text), gitUrl (text), defaultBranch (text), credential (password)
		// Switch is a button not an input, so inputs are: repoId, gitUrl, defaultBranch, credential
		const [repoIdInput, gitUrlInput, branchInput, credInput] = Array.from(allInputs);

		fireEvent.change(repoIdInput, { target: { value: 'new-repo' } });
		fireEvent.change(gitUrlInput, { target: { value: 'https://github.com/org/new' } });
		fireEvent.change(branchInput, { target: { value: 'main' } });
		fireEvent.change(credInput, { target: { value: 'my-pat-token' } });

		// submit via OK button
		const okBtn = screen.getByRole('button', { name: /ok/i });
		fireEvent.click(okBtn);

		await waitFor(() => {
			expect(mockUpsertRepo).toHaveBeenCalledTimes(1);
			const payload = mockUpsertRepo.mock.calls[0][0];
			// credential sent verbatim for new repo
			expect(payload.credential).toBe('my-pat-token');
			expect(payload.contractVersion).toBe('ds.codebase_repo.v1');
		});
	});

	// Case 3b: edit existing repo without entering credential → CREDENTIAL_UNCHANGED sent
	it('edits an existing repo without credential and sends CREDENTIAL_UNCHANGED', async () => {
		mockListRepos.mockResolvedValue({ data: [defaultRepo] });
		mockListRepos
			.mockResolvedValueOnce({ data: [defaultRepo] })
			.mockResolvedValue({ data: [defaultRepo] });

		render(<CodeRcaSettings />);

		// wait for repo row to appear and find Edit button
		const editBtn = await screen.findByRole('button', { name: /^Edit$/i });
		fireEvent.click(editBtn);

		await waitFor(() => {
			expect(screen.getByRole('dialog')).toBeInTheDocument();
		});

		// do NOT fill in credential — leave it blank
		// submit
		const okBtn = screen.getByRole('button', { name: /ok/i });
		fireEvent.click(okBtn);

		await waitFor(() => {
			expect(mockUpsertRepo).toHaveBeenCalledTimes(1);
			const payload = mockUpsertRepo.mock.calls[0][0];
			expect(payload.credential).toBe('<unchanged>');
		});
	});

	// Case 4: switch to runs tab → listRuns called + status tag rendered
	it('switches to runs tab, calls listRuns, and renders a status tag', async () => {
		render(<CodeRcaSettings />);

		// wait for tabs to render
		const runsTab = await screen.findByText('tab_runs');
		fireEvent.click(runsTab);

		await waitFor(() => {
			expect(mockListRuns).toHaveBeenCalled();
		});

		// status tag for 'done' run should be in the DOM
		await waitFor(() => {
			expect(screen.getByText('done')).toBeInTheDocument();
		});
	});

	// Case 5: run row click → getRun called + drawer shows rootCause/proposedFix
	it('clicks a run row and shows rootCause and proposedFix in drawer', async () => {
		render(<CodeRcaSettings />);

		// navigate to runs tab
		const runsTab = await screen.findByText('tab_runs');
		fireEvent.click(runsTab);

		// wait for run row
		const statusTag = await screen.findByText('done');
		// click the row (the tag is inside the row)
		fireEvent.click(statusTag.closest('tr') ?? statusTag);

		await waitFor(() => {
			expect(mockGetRun).toHaveBeenCalledWith('run-1');
		});

		// drawer should show rootCause and proposedFix
		await waitFor(() => {
			expect(screen.getByText('NullPointerException in line 42')).toBeInTheDocument();
			expect(screen.getByText('Add null check before usage')).toBeInTheDocument();
		});

		// run_hitl_notice key rendered
		expect(screen.getByText('run_hitl_notice')).toBeInTheDocument();
	});
});
