import { parseSopRows } from '../parseSopExcel';

const VALID_ROW = {
	sop_id: 'SOP-PAY-001',
	title: 'Payment API 5xx',
	version: '2026-06-01.1',
	owner_team: 'payments',
	approval_status: 'approved',
	source_id: 'src-managed-markdown-default',
	project_ids: 'customer-a',
	environments: 'prod',
	display_url: 'https://kb.example/sop/SOP-PAY-001',
	tags: 'payment-api,critical',
	service_account_profile: 'managed-markdown-local',
	body_markdown: '# Payment API 5xx\n\n1. Check logs',
};

describe('parseSopRows', () => {
	it('parses a valid row into a SopDocument', () => {
		const result = parseSopRows([VALID_ROW]);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(0);
		expect(result.rows[0].valid).toBe(true);
		expect(result.rows[0].document?.sopId).toBe('SOP-PAY-001');
		expect(result.rows[0].document?.approvalStatus).toBe('approved');
		expect(result.rows[0].document?.tenantScope.projectIds).toEqual(['customer-a']);
		expect(result.rows[0].document?.tags).toEqual(['payment-api', 'critical']);
		expect(result.rows[0].document?.checksum).toMatch(/^sha256:/);
	});

	it('applies defaults for optional fields when empty', () => {
		const row = { ...VALID_ROW, approval_status: '', source_id: '', service_account_profile: '' };
		const result = parseSopRows([row]);
		expect(result.rows[0].valid).toBe(true);
		expect(result.rows[0].document?.approvalStatus).toBe('approved');
		expect(result.rows[0].document?.source.sourceId).toBe('src-managed-markdown-default');
		expect(result.rows[0].document?.securityContext.serviceAccountProfile).toBe('managed-markdown-local');
	});

	it('marks a row with missing required field as error', () => {
		const row = { ...VALID_ROW, sop_id: '' };
		const result = parseSopRows([row]);
		expect(result.validCount).toBe(0);
		expect(result.errorCount).toBe(1);
		expect(result.rows[0].valid).toBe(false);
		expect(result.rows[0].error).toContain('sop_id');
	});

	it('marks a row with invalid approval_status as error', () => {
		const row = { ...VALID_ROW, approval_status: 'invalid-status' };
		const result = parseSopRows([row]);
		expect(result.rows[0].valid).toBe(false);
		expect(result.rows[0].error).toContain('approval_status');
	});

	it('handles mixed valid and invalid rows', () => {
		const invalidRow = { ...VALID_ROW, sop_id: '', version: '' };
		const result = parseSopRows([VALID_ROW, invalidRow]);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(1);
	});

	it('returns empty result for empty rows array', () => {
		const result = parseSopRows([]);
		expect(result.validCount).toBe(0);
		expect(result.errorCount).toBe(0);
		expect(result.rows).toHaveLength(0);
	});
});
