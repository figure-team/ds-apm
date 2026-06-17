import * as XLSX from 'xlsx';
import SHA256 from 'crypto-js/sha256';
import {
	SOP_DOCUMENT_CONTRACT_VERSION,
	type SopApprovalStatus,
	type SopDocument,
} from 'api/v2/rules/sopDocuments';

const REQUIRED_COLUMNS = [
	'sop_id',
	'title',
	'version',
	'owner_team',
	'project_ids',
	'environments',
	'body_markdown',
] as const;

const ALLOWED_APPROVAL_STATUSES: SopApprovalStatus[] = [
	'approved',
	'draft',
	'deprecated',
	'disabled',
];

export type ParsedSopRow = {
	rowIndex: number;
	valid: boolean;
	error?: string;
	document?: SopDocument;
	raw: Record<string, string>;
};

export type ParseSopExcelResult = {
	rows: ParsedSopRow[];
	validCount: number;
	errorCount: number;
};

function parseTags(value: string): string[] {
	if (!value) return [];
	return value
		.split(',')
		.map((t) => t.trim())
		.filter(Boolean);
}

function checksumForMarkdown(bodyMarkdown: string): string {
	return `sha256:${SHA256(bodyMarkdown).toString()}`;
}

export function parseSopRows(rows: Record<string, string>[]): ParseSopExcelResult {
	const parsed: ParsedSopRow[] = rows.map((row, idx) => {
		const missingFields = REQUIRED_COLUMNS.filter(
			(col) => !String(row[col] ?? '').trim(),
		);
		if (missingFields.length > 0) {
			return {
				rowIndex: idx + 2,
				valid: false,
				error: `필수 필드 누락: ${missingFields.join(', ')}`,
				raw: row,
			};
		}

		const approvalStatus = (
			String(row.approval_status ?? '').trim() || 'approved'
		) as SopApprovalStatus;
		if (!ALLOWED_APPROVAL_STATUSES.includes(approvalStatus)) {
			return {
				rowIndex: idx + 2,
				valid: false,
				error: `approval_status 허용 값: ${ALLOWED_APPROVAL_STATUSES.join(', ')}`,
				raw: row,
			};
		}

		const bodyMarkdown = String(row.body_markdown ?? '').trim();
		const customerUpdateTemplate = String(
			row.customer_update_template ?? '',
		).trim();
		const vendorRequestTemplate = String(
			row.vendor_request_template ?? '',
		).trim();
		const document: SopDocument = {
			contractVersion: SOP_DOCUMENT_CONTRACT_VERSION,
			sopId: String(row.sop_id).trim(),
			title: String(row.title).trim(),
			version: String(row.version).trim(),
			checksum: checksumForMarkdown(bodyMarkdown),
			source: {
				type: 'managed_markdown',
				sourceId:
					String(row.source_id ?? '').trim() || 'src-managed-markdown-default',
			},
			bodyMarkdown,
			customerUpdateTemplate: customerUpdateTemplate || undefined,
			vendorRequestTemplate: vendorRequestTemplate || undefined,
			displayUrl: String(row.display_url ?? '').trim() || undefined,
			ownerTeam: String(row.owner_team).trim(),
			approvalStatus,
			tenantScope: {
				projectIds: parseTags(String(row.project_ids ?? '')),
				environments: parseTags(String(row.environments ?? '')),
			},
			tags: parseTags(String(row.tags ?? '')),
			updatedAt: new Date().toISOString(),
			securityContext: {
				serviceAccountProfile:
					String(row.service_account_profile ?? '').trim() ||
					'managed-markdown-local',
				secretRefVisible: false,
				browserCredentialsUsed: false,
				redactionApplied: true,
			},
		};

		return { rowIndex: idx + 2, valid: true, document, raw: row };
	});

	return {
		rows: parsed,
		validCount: parsed.filter((r) => r.valid).length,
		errorCount: parsed.filter((r) => !r.valid).length,
	};
}

export function parseSopExcel(file: File): Promise<ParseSopExcelResult> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = (e): void => {
			try {
				const data = e.target?.result;
				const workbook = XLSX.read(data, { type: 'binary' });
				const sheetName = workbook.SheetNames[0];
				const sheet = workbook.Sheets[sheetName];
				const rows = XLSX.utils.sheet_to_json<Record<string, string>>(sheet, {
					raw: false,
				});

				if (rows.length === 0) {
					resolve({ rows: [], validCount: 0, errorCount: 0 });
					return;
				}

				const missingColumns = REQUIRED_COLUMNS.filter(
					(col) => !(col in rows[0]),
				);
				if (missingColumns.length > 0) {
					reject(
						new Error(`필수 컬럼 누락: ${missingColumns.join(', ')}`),
					);
					return;
				}

				resolve(parseSopRows(rows));
			} catch (err) {
				reject(err instanceof Error ? err : new Error('파일 파싱 실패'));
			}
		};
		reader.onerror = (): void => reject(new Error('파일 읽기 실패'));
		reader.readAsBinaryString(file);
	});
}

export function downloadSopExcelTemplate(): void {
	const headers = [
		'sop_id',
		'title',
		'version',
		'owner_team',
		'approval_status',
		'source_id',
		'project_ids',
		'environments',
		'display_url',
		'tags',
		'service_account_profile',
		'body_markdown',
		'customer_update_template',
		'vendor_request_template',
	];
	const example = [
		'SOP-PAY-001',
		'Payment API 5xx response',
		'2026-06-01.1',
		'payments',
		'approved',
		'src-managed-markdown-default',
		'customer-a',
		'prod',
		'https://kb.example/sop/SOP-PAY-001',
		'payment-api,critical',
		'managed-markdown-local',
		'# Payment API 5xx response\n\n1. Check payment dashboard\n2. Inspect PG timeout logs',
		'[안내] {증상} 발생. 영향: {범위}. 조치: {조치}. 문의: 고객센터',
		'안녕하세요. {서비스}에서 {증상} 확인됩니다. {확인 요청 항목} 확인 부탁드립니다.',
	];

	const wb = XLSX.utils.book_new();
	const ws = XLSX.utils.aoa_to_sheet([headers, example]);
	ws['!cols'] = [
		{ wch: 20 }, // sop_id
		{ wch: 35 }, // title
		{ wch: 15 }, // version
		{ wch: 20 }, // owner_team
		{ wch: 18 }, // approval_status
		{ wch: 30 }, // source_id
		{ wch: 20 }, // project_ids
		{ wch: 15 }, // environments
		{ wch: 40 }, // display_url
		{ wch: 25 }, // tags
		{ wch: 25 }, // service_account_profile
		{ wch: 60 }, // body_markdown
		{ wch: 45 }, // customer_update_template
		{ wch: 45 }, // vendor_request_template
	];
	XLSX.utils.book_append_sheet(wb, ws, 'SOP Template');
	XLSX.writeFile(wb, 'sop-template.xlsx');
}
