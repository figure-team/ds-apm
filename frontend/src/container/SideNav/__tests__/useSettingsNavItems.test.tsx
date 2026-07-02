import { renderHook } from '@testing-library/react';
import { useSettingsNavItems } from '../useSettingsNavItems';

// Mutable context objects — each test can mutate these before renderHook
const mockAppContext: any = {
	user: { role: 'ADMIN' },
	featureFlags: [],
	trialInfo: { workSpaceBlock: false },
	isFetchingActiveLicense: false,
};
const mockLicense: any = {
	isCloudUser: false,
	isEnterpriseSelfHostedUser: false,
	isCommunityEnterpriseUser: false,
};

jest.mock('providers/App/App', () => ({
	useAppContext: (): any => mockAppContext,
}));
jest.mock('hooks/useGetTenantLicense', () => ({
	useGetTenantLicense: (): any => mockLicense,
}));

// Helper: returns itemKeys of all enabled items across all sections
const enabledKeys = (sections: ReturnType<typeof useSettingsNavItems>): string[] =>
	sections
		.flatMap((s) => s.items)
		.filter((i) => i.isEnabled)
		.map((i) => i.itemKey as string);

beforeEach(() => {
	// Reset to community-admin baseline before each test
	mockAppContext.user = { role: 'ADMIN' };
	mockAppContext.trialInfo = { workSpaceBlock: false };
	mockAppContext.isFetchingActiveLicense = false;
	mockLicense.isCloudUser = false;
	mockLicense.isEnterpriseSelfHostedUser = false;
	mockLicense.isCommunityEnterpriseUser = false;
});

describe('useSettingsNavItems', () => {
	it('always enables Account and keeps section structure', () => {
		const { result } = renderHook(() => useSettingsNavItems());
		const allItems = result.current.flatMap((s) => s.items);
		const account = allItems.find((i) => i.itemKey === 'account');
		expect(account?.isEnabled).toBe(true);
		// authentication section is always preserved
		expect(result.current.some((s) => s.key === 'authentication')).toBe(true);
	});

	it('Cloud Admin: enables billing, roles, members, service-accounts, integrations, ingestion, sso, mcp-server, ai-module, code-rca, incident-report, account', () => {
		mockLicense.isCloudUser = true;
		mockAppContext.user = { role: 'ADMIN' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		// Must be enabled
		expect(keys).toContain('billing');
		expect(keys).toContain('roles');
		expect(keys).toContain('members');
		expect(keys).toContain('service-accounts');
		expect(keys).toContain('integrations');
		expect(keys).toContain('ingestion');
		expect(keys).toContain('sso');
		expect(keys).toContain('mcp-server');
		expect(keys).toContain('ai-module');
		expect(keys).toContain('code-rca');
		expect(keys).toContain('incident-report');
		expect(keys).toContain('account');
		// Remediation Targets is Admin-only (spec §4.1)
		expect(keys).toContain('remediation-targets');
	});

	it('Cloud Viewer: billing and roles are NOT enabled; account IS enabled', () => {
		mockLicense.isCloudUser = true;
		mockAppContext.user = { role: 'VIEWER' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).not.toContain('billing');
		expect(keys).not.toContain('roles');
		// mcp-server is editor/admin-only
		expect(keys).not.toContain('mcp-server');
		expect(keys).toContain('account');
	});

	it('Cloud Editor: enables ingestion, integrations, mcp-server, ai-module, code-rca, incident-report; billing NOT enabled', () => {
		mockLicense.isCloudUser = true;
		mockAppContext.user = { role: 'EDITOR' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).toContain('ingestion');
		expect(keys).toContain('integrations');
		expect(keys).toContain('mcp-server');
		expect(keys).toContain('ai-module');
		expect(keys).toContain('code-rca');
		expect(keys).toContain('incident-report');
		expect(keys).not.toContain('billing');
		// Remediation Targets must NOT open to Editor (spec §4.1)
		expect(keys).not.toContain('remediation-targets');
	});

	it('Self-hosted Admin: enables billing, roles, members, service-accounts, integrations, sso, ingestion, mcp-server, ai-module, code-rca, incident-report', () => {
		mockLicense.isEnterpriseSelfHostedUser = true;
		mockAppContext.user = { role: 'ADMIN' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).toContain('billing');
		expect(keys).toContain('roles');
		expect(keys).toContain('members');
		expect(keys).toContain('service-accounts');
		expect(keys).toContain('integrations');
		expect(keys).toContain('sso');
		expect(keys).toContain('ingestion');
		expect(keys).toContain('mcp-server');
		expect(keys).toContain('ai-module');
		expect(keys).toContain('code-rca');
		expect(keys).toContain('incident-report');
		// Remediation Targets is Admin-only (spec §4.1)
		expect(keys).toContain('remediation-targets');
	});

	it('Self-hosted Editor: enables integrations, ingestion, mcp-server, ai-module, code-rca, incident-report; billing NOT enabled', () => {
		mockLicense.isEnterpriseSelfHostedUser = true;
		mockAppContext.user = { role: 'EDITOR' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).toContain('integrations');
		expect(keys).toContain('ingestion');
		expect(keys).toContain('mcp-server');
		expect(keys).toContain('ai-module');
		expect(keys).toContain('code-rca');
		expect(keys).toContain('incident-report');
		expect(keys).not.toContain('billing');
		// Remediation Targets must NOT open to Editor (spec §4.1)
		expect(keys).not.toContain('remediation-targets');
	});

	it('Community Admin (all license flags false): enables sso, members, service-accounts, roles, ai-module, code-rca, incident-report; billing and integrations NOT enabled', () => {
		// all license flags already false from beforeEach
		mockAppContext.user = { role: 'ADMIN' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).toContain('sso');
		expect(keys).toContain('members');
		expect(keys).toContain('service-accounts');
		expect(keys).toContain('roles');
		expect(keys).toContain('ai-module');
		expect(keys).toContain('code-rca');
		expect(keys).toContain('incident-report');
		expect(keys).not.toContain('billing');
		expect(keys).not.toContain('integrations');
		// Remediation Targets is Admin-only (spec §4.1)
		expect(keys).toContain('remediation-targets');
	});

	it('Community Editor: remediation-targets NOT enabled (Admin-only, spec §4.1)', () => {
		// all license flags false from beforeEach; community has no isEditor branch
		mockAppContext.user = { role: 'EDITOR' };

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).not.toContain('remediation-targets');
	});

	it('workSpaceBlock=true: only billing, sso, members, account (and keyboard-shortcuts) are enabled for admin', () => {
		mockAppContext.user = { role: 'ADMIN' };
		mockAppContext.trialInfo = { workSpaceBlock: true };
		mockAppContext.isFetchingActiveLicense = false;

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).toContain('billing');
		expect(keys).toContain('sso');
		expect(keys).toContain('members');
		expect(keys).toContain('account');

		// Items that should be disabled under workspace block
		expect(keys).not.toContain('roles');
		expect(keys).not.toContain('service-accounts');
		expect(keys).not.toContain('integrations');
		expect(keys).not.toContain('ingestion');
		expect(keys).not.toContain('mcp-server');
		expect(keys).not.toContain('ai-module');
		expect(keys).not.toContain('code-rca');
		expect(keys).not.toContain('incident-report');
		// workspace/notification-channels/sop-documents are also disabled under workSpaceBlock
		expect(keys).not.toContain('workspace');
		expect(keys).not.toContain('notification-channels');
		expect(keys).not.toContain('sop-documents');
	});

	it('isCommunityEnterpriseUser=true: manage-license IS enabled', () => {
		mockLicense.isCommunityEnterpriseUser = true;

		const { result } = renderHook(() => useSettingsNavItems());
		const keys = enabledKeys(result.current);

		expect(keys).toContain('manage-license');
	});
});
