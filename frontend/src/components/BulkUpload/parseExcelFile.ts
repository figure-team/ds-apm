import * as XLSX from 'xlsx';

import type { BulkParseResult, BulkRowBase } from './types';

/**
 * .xlsx 파일을 읽어 첫 시트를 행 객체 배열로 만든 뒤 도메인 파서에 넘긴다.
 * 필수 컬럼이 헤더에 없으면 파싱 자체를 거부한다(행 단위 오류가 아니라 파일 단위 오류).
 */
export function parseExcelFile<TRow extends BulkRowBase>(
	file: File,
	requiredColumns: readonly string[],
	parseRows: (rows: Record<string, string>[]) => BulkParseResult<TRow>,
): Promise<BulkParseResult<TRow>> {
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

				const missingColumns = requiredColumns.filter((col) => !(col in rows[0]));
				if (missingColumns.length > 0) {
					reject(new Error(`필수 컬럼 누락: ${missingColumns.join(', ')}`));
					return;
				}

				resolve(parseRows(rows));
			} catch (err) {
				reject(err instanceof Error ? err : new Error('파일 파싱 실패'));
			}
		};
		reader.onerror = (): void => reject(new Error('파일 읽기 실패'));
		reader.readAsBinaryString(file);
	});
}

export type ExcelTemplateSpec = {
	headers: string[];
	/** 헤더 아래에 넣을 예시 행들. */
	exampleRows: string[][];
	/** headers와 같은 길이의 컬럼 폭(wch). */
	columnWidths: number[];
	sheetName: string;
	fileName: string;
};

/** 헤더 + 예시 행으로 구성된 업로드 양식 .xlsx를 내려받는다. */
export function downloadExcelTemplate(spec: ExcelTemplateSpec): void {
	const wb = XLSX.utils.book_new();
	const ws = XLSX.utils.aoa_to_sheet([spec.headers, ...spec.exampleRows]);
	ws['!cols'] = spec.columnWidths.map((wch) => ({ wch }));
	XLSX.utils.book_append_sheet(wb, ws, spec.sheetName);
	XLSX.writeFile(wb, spec.fileName);
}
