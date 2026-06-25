import * as XLSX from 'xlsx';

import type { Runbook, RunbookStatus } from './types';

const REQUIRED_COLUMNS = ['title', 'executable_script'] as const;

// 일괄 업로드는 신규 등록이므로 deprecated는 허용하지 않는다.
const ALLOWED_STATUSES: RunbookStatus[] = ['draft', 'approved'];

export type ParsedRunbookRow = {
	rowIndex: number;
	valid: boolean;
	error?: string;
	runbook?: Partial<Runbook>;
	raw: Record<string, string>;
};

export type ParseRunbookExcelResult = {
	rows: ParsedRunbookRow[];
	validCount: number;
	errorCount: number;
};

export function parseRunbookRows(
	rows: Record<string, string>[],
): ParseRunbookExcelResult {
	const parsed: ParsedRunbookRow[] = rows.map((row, idx) => {
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

		// 안전 기본값: 미지정 시 draft (approved는 실제 bash 실행 대상).
		const status = (String(row.status ?? '').trim() || 'draft') as RunbookStatus;
		if (!ALLOWED_STATUSES.includes(status)) {
			return {
				rowIndex: idx + 2,
				valid: false,
				error: `status 허용 값: ${ALLOWED_STATUSES.join(', ')}`,
				raw: row,
			};
		}

		const runbook: Partial<Runbook> = {
			title: String(row.title).trim(),
			description: String(row.description ?? '').trim(),
			executableScript: String(row.executable_script).trim(),
			status,
		};

		return { rowIndex: idx + 2, valid: true, runbook, raw: row };
	});

	return {
		rows: parsed,
		validCount: parsed.filter((r) => r.valid).length,
		errorCount: parsed.filter((r) => !r.valid).length,
	};
}

export function parseRunbookExcel(file: File): Promise<ParseRunbookExcelResult> {
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
					reject(new Error(`필수 컬럼 누락: ${missingColumns.join(', ')}`));
					return;
				}

				resolve(parseRunbookRows(rows));
			} catch (err) {
				reject(err instanceof Error ? err : new Error('파일 파싱 실패'));
			}
		};
		reader.onerror = (): void => reject(new Error('파일 읽기 실패'));
		reader.readAsBinaryString(file);
	});
}

export function downloadRunbookExcelTemplate(): void {
	const headers = ['title', 'description', 'executable_script', 'status'];
	const examples = [
		[
			'토스페이 결제 API 헬스 점검',
			'토스페이 PG 엔드포인트의 응답 코드·지연을 확인한다. 시크릿/토큰은 출력 금지.',
			'#!/usr/bin/env bash\nset -euo pipefail\nURL="${TOSSPAY_HEALTH_URL:-https://pay.toss.im/health}"\ncode=$(curl -s -o /dev/null -w \'%{http_code}\' --max-time 5 "$URL" || echo 000)\necho "tosspay health: http=${code} url=${URL}"\ncase "$code" in 2*|3*) echo OK;; *) echo FAIL; exit 1;; esac',
			'draft',
		],
		[
			'결제 에러 로그 급증 점검',
			'최근 결제 로그에서 5xx/timeout 발생 건수를 집계한다.',
			'#!/usr/bin/env bash\nset -euo pipefail\nLOG="${PAYMENT_LOG_PATH:-/var/log/payment/app.log}"\n[ -r "$LOG" ] || { echo "log not readable: $LOG"; exit 1; }\nn=$(grep -cE \'TOSSPAY.*(5[0-9]{2}|timeout)\' "$LOG" || true)\necho "tosspay 5xx/timeout count: ${n}"\n[ "$n" -lt 50 ] || exit 1',
			'draft',
		],
	];

	const wb = XLSX.utils.book_new();
	const ws = XLSX.utils.aoa_to_sheet([headers, ...examples]);
	ws['!cols'] = [
		{ wch: 30 }, // title
		{ wch: 45 }, // description
		{ wch: 70 }, // executable_script
		{ wch: 12 }, // status
	];
	XLSX.utils.book_append_sheet(wb, ws, 'Runbook Template');
	XLSX.writeFile(wb, 'runbook-template.xlsx');
}
