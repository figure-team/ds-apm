import { parseRunbookRows } from './parseRunbookExcel';

const VALID_ROW = {
	title: '토스페이 결제 API 헬스 점검',
	description: '토스페이 PG 엔드포인트 응답 확인',
	executable_script: '#!/usr/bin/env bash\necho ok',
	status: 'approved',
};

describe('parseRunbookRows', () => {
	it('parses a valid row into a Runbook payload', () => {
		const result = parseRunbookRows([VALID_ROW]);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(0);
		expect(result.rows[0].valid).toBe(true);
		expect(result.rows[0].runbook?.title).toBe('토스페이 결제 API 헬스 점검');
		expect(result.rows[0].runbook?.executableScript).toContain('#!/usr/bin/env bash');
		expect(result.rows[0].runbook?.status).toBe('approved');
	});

	it('defaults status to draft when empty', () => {
		const row = { ...VALID_ROW, status: '' };
		const result = parseRunbookRows([row]);
		expect(result.rows[0].valid).toBe(true);
		expect(result.rows[0].runbook?.status).toBe('draft');
	});

	it('marks a row with missing required field as error', () => {
		const row = { ...VALID_ROW, executable_script: '' };
		const result = parseRunbookRows([row]);
		expect(result.validCount).toBe(0);
		expect(result.errorCount).toBe(1);
		expect(result.rows[0].valid).toBe(false);
		expect(result.rows[0].error).toContain('executable_script');
	});

	it('rejects status values outside draft/approved', () => {
		const row = { ...VALID_ROW, status: 'deprecated' };
		const result = parseRunbookRows([row]);
		expect(result.rows[0].valid).toBe(false);
		expect(result.rows[0].error).toContain('status');
	});

	it('handles mixed valid and invalid rows', () => {
		const invalidRow = { ...VALID_ROW, title: '' };
		const result = parseRunbookRows([VALID_ROW, invalidRow]);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(1);
		// rowIndex는 헤더 행을 고려해 2부터 시작한다.
		expect(result.rows[1].rowIndex).toBe(3);
	});

	it('returns empty result for empty rows array', () => {
		const result = parseRunbookRows([]);
		expect(result.validCount).toBe(0);
		expect(result.errorCount).toBe(0);
		expect(result.rows).toHaveLength(0);
	});
});
