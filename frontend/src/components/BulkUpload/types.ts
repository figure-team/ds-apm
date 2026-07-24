/**
 * 일괄(.xlsx) 업로드 흐름의 공용 타입.
 * SOP 문서와 Runbook이 같은 흐름(파싱 → 미리보기 → 일괄 등록)을 쓰므로
 * 행 형태만 도메인별로 확장하고 나머지는 공유한다.
 */

/** 파싱된 한 행의 공통 부분. 도메인 payload(document/runbook 등)는 교차 타입으로 얹는다. */
export type BulkRowBase = {
	/** 시트의 1-based 행 번호(헤더 행 포함이라 데이터 첫 행이 2). */
	rowIndex: number;
	valid: boolean;
	error?: string;
	raw: Record<string, string>;
};

export type BulkParseResult<TRow extends BulkRowBase> = {
	rows: TRow[];
	validCount: number;
	errorCount: number;
};

/** 서버 등록 후 행별 결과. 도메인 응답 타입이 이 형태의 상위형이면 그대로 대입된다. */
export type BulkRowResult = {
	status: 'ok' | 'error';
	error?: string;
};

export type BulkRowWithResult<TRow extends BulkRowBase> = TRow & {
	result?: BulkRowResult;
};

/** 파싱된 행 배열에 유효/오류 건수를 붙여 반환한다. */
export function summarizeBulkRows<TRow extends BulkRowBase>(
	rows: TRow[],
): BulkParseResult<TRow> {
	return {
		rows,
		validCount: rows.filter((r) => r.valid).length,
		errorCount: rows.filter((r) => !r.valid).length,
	};
}
