import * as XLSX from 'xlsx';

import { downloadExcelTemplate, parseExcelFile } from '../parseExcelFile';
import {
	type BulkParseResult,
	type BulkRowBase,
	summarizeBulkRows,
} from '../types';

// writeFile은 재정의 불가 속성이라 spyOn이 안 먹는다 — 모듈 단위로 갈아끼운다.
jest.mock('xlsx', () => {
	const actual = jest.requireActual('xlsx');
	return { ...actual, writeFile: jest.fn() };
});

type Row = BulkRowBase;

const REQUIRED = ['title', 'script'] as const;

function makeXlsxFile(aoa: string[][]): File {
	const wb = XLSX.utils.book_new();
	XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet(aoa), 'Sheet1');
	const buf = XLSX.write(wb, { bookType: 'xlsx', type: 'array' });
	return new File([buf], 'upload.xlsx');
}

function parseRows(rows: Record<string, string>[]): BulkParseResult<Row> {
	return summarizeBulkRows<Row>(
		rows.map((raw, idx) => ({
			rowIndex: idx + 2,
			valid: Boolean(raw.title),
			error: raw.title ? undefined : '필수 필드 누락: title',
			raw,
		})),
	);
}

describe('summarizeBulkRows', () => {
	it('유효/오류 건수를 센다', () => {
		const result = summarizeBulkRows<Row>([
			{ rowIndex: 2, valid: true, raw: {} },
			{ rowIndex: 3, valid: false, raw: {} },
			{ rowIndex: 4, valid: true, raw: {} },
		]);
		expect(result.validCount).toBe(2);
		expect(result.errorCount).toBe(1);
		expect(result.rows).toHaveLength(3);
	});
});

describe('parseExcelFile', () => {
	it('헤더에 필수 컬럼이 없으면 파일 단위로 거부한다', async () => {
		const file = makeXlsxFile([
			['title', 'note'],
			['a', 'b'],
		]);
		await expect(parseExcelFile(file, REQUIRED, parseRows)).rejects.toThrow(
			'필수 컬럼 누락: script',
		);
	});

	it('데이터 행이 없으면 빈 결과를 돌려준다', async () => {
		const file = makeXlsxFile([['title', 'script']]);
		await expect(
			parseExcelFile(file, REQUIRED, parseRows),
		).resolves.toStrictEqual({
			rows: [],
			validCount: 0,
			errorCount: 0,
		});
	});

	it('행 파싱을 도메인 파서에 위임하고 rowIndex는 헤더를 감안해 2부터 센다', async () => {
		const file = makeXlsxFile([
			['title', 'script'],
			['ok', 'echo 1'],
			['', 'echo 2'],
		]);
		const result = await parseExcelFile(file, REQUIRED, parseRows);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(1);
		expect(result.rows[0].rowIndex).toBe(2);
		expect(result.rows[1].error).toBe('필수 필드 누락: title');
	});
});

describe('downloadExcelTemplate', () => {
	it('헤더·예시 행·컬럼 폭을 시트에 실어 파일명으로 내려받는다', () => {
		const appendSpy = jest.spyOn(XLSX.utils, 'book_append_sheet');
		const writeMock = XLSX.writeFile as jest.Mock;
		writeMock.mockClear();

		downloadExcelTemplate({
			headers: ['title', 'script'],
			exampleRows: [['예시', 'echo 1']],
			columnWidths: [30, 70],
			sheetName: 'Sheet',
			fileName: 'template.xlsx',
		});

		const sheet = appendSpy.mock.calls[0][1] as XLSX.WorkSheet;
		expect(sheet['!cols']).toStrictEqual([{ wch: 30 }, { wch: 70 }]);
		expect(XLSX.utils.sheet_to_json(sheet, { raw: false })).toStrictEqual([
			{ title: '예시', script: 'echo 1' },
		]);
		expect(writeMock.mock.calls[0][1]).toBe('template.xlsx');

		appendSpy.mockRestore();
	});
});
