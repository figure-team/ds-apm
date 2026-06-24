import type { SopDocumentSummary } from 'api/v2/rules/sopDocuments';

import {
	hasSopBinding,
	resolveSopBindingDocument,
	validateSopAnnotations,
	validateSopAnnotationValue,
	validateSopLabelValue,
} from '../sopMetadata';

const makeSopDoc = (
	overrides: Partial<SopDocumentSummary>,
): SopDocumentSummary =>
	({
		sopId: 'SOP-PAY-001',
		title: 'Payment API 5xx response',
		version: '2026-04-20.1',
		approvalStatus: 'approved',
		ownerTeam: 'payments-team',
		source: { type: 'managed_markdown', sourceId: 'confluence' },
		tenantScope: { projectIds: [], environments: [] },
		...overrides,
	} as SopDocumentSummary);

describe('sopMetadata', () => {
	it('detects SOP binding from sop_id labels or sop_url annotations', () => {
		expect(hasSopBinding({}, {})).toBe(false);
		expect(hasSopBinding({ sop_id: 'SOP-PAY-001' }, {})).toBe(true);
		expect(
			hasSopBinding({}, { sop_url: 'https://kb.example/sop/SOP-PAY-001' }),
		).toBe(true);
	});

	it('does not warn for empty or normal SOP metadata values', () => {
		expect(validateSopLabelValue(undefined)).toStrictEqual([]);
		expect(validateSopLabelValue('SOP-PAY-001')).toStrictEqual([]);
		expect(
			validateSopAnnotationValue('sop_url', 'https://kb.example/sop/SOP-PAY-001'),
		).toStrictEqual([]);
		expect(
			validateSopAnnotationValue('sop_version', '2026-04-20.3'),
		).toStrictEqual([]);
		expect(validateSopAnnotationValue('sop_source', 'confluence')).toStrictEqual(
			[],
		);
	});

	it('warns when SOP labels are too long or secret-like', () => {
		expect(validateSopLabelValue(`SOP-${'a'.repeat(121)}`)).toStrictEqual([
			'Keep sop_id under 120 characters.',
		]);
		expect(validateSopLabelValue('token=do-not-store-this')).toStrictEqual([
			'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
		]);
	});

	it('warns for unsafe SOP annotation values', () => {
		expect(
			validateSopAnnotationValue('sop_url', 'javascript:alert(1)'),
		).toStrictEqual(['Use an http:// or https:// SOP URL.']);
		expect(
			validateSopAnnotationValue(
				'sop_url',
				'https://kb.example/sop/SOP-PAY-001?token=hidden',
			),
		).toStrictEqual([
			'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
		]);
		expect(
			validateSopAnnotationValue(
				'sop_url',
				'https://user:pass@kb.example/sop/SOP-PAY-001',
			),
		).toStrictEqual([
			'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
		]);
		expect(
			validateSopAnnotationValue(
				'sop_title',
				'Authorization: Bearer abcdefghijklmnop',
			),
		).toStrictEqual([
			'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
		]);
	});

	it('returns warnings keyed by SOP annotation', () => {
		expect(
			validateSopAnnotations({
				sop_title: 'Payment API incident SOP',
				sop_url: 'ftp://kb.example/sop/SOP-PAY-001',
				sop_version: 'token=do-not-store-this',
			}),
		).toStrictEqual({
			sop_url: ['Use an http:// or https:// SOP URL.'],
			sop_version: [
				'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
			],
		});
	});

	describe('resolveSopBindingDocument', () => {
		it('returns undefined for an empty or unknown SOP_ID', () => {
			expect(resolveSopBindingDocument([], '')).toBeUndefined();
			expect(
				resolveSopBindingDocument(
					[makeSopDoc({ version: '2026-04-20.1' })],
					'SOP-UNKNOWN',
				),
			).toBeUndefined();
		});

		it('ignores non-approved documents even if the SOP_ID matches', () => {
			const docs = [
				makeSopDoc({ version: '2026-04-20.5', approvalStatus: 'draft' }),
				makeSopDoc({ version: '2026-04-20.4', approvalStatus: 'deprecated' }),
				makeSopDoc({ version: '2026-04-20.3', approvalStatus: 'disabled' }),
			];

			expect(resolveSopBindingDocument(docs, 'SOP-PAY-001')).toBeUndefined();
		});

		it('picks the approved document, skipping an earlier non-approved match', () => {
			const docs = [
				makeSopDoc({ version: '2026-04-20.9', approvalStatus: 'draft' }),
				makeSopDoc({ version: '2026-04-20.2', approvalStatus: 'approved' }),
			];

			const match = resolveSopBindingDocument(docs, 'SOP-PAY-001');

			expect(match?.version).toBe('2026-04-20.2');
			expect(match?.approvalStatus).toBe('approved');
		});

		it('picks the latest version among approved documents', () => {
			const docs = [
				makeSopDoc({ version: '2026-04-20.1', approvalStatus: 'approved' }),
				makeSopDoc({ version: '2026-04-20.3', approvalStatus: 'approved' }),
				makeSopDoc({ version: '2026-04-20.2', approvalStatus: 'approved' }),
			];

			expect(resolveSopBindingDocument(docs, 'SOP-PAY-001')?.version).toBe(
				'2026-04-20.3',
			);
		});
	});
});
