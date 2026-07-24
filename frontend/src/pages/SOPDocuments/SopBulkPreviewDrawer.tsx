import { useCallback } from 'react';
import type { ColumnsType } from 'antd/es/table';
import {
	createSopDocumentBatch,
	SOP_DOCUMENT_LIST_CONTRACT_VERSION,
} from 'api/v2/rules/sopDocuments';
import BulkPreviewDrawer from 'components/BulkUpload/BulkPreviewDrawer';
import type {
	BulkRowResult,
	BulkRowWithResult,
} from 'components/BulkUpload/types';
import type { ParseSopExcelResult, ParsedSopRow } from './parseSopExcel';

type Props = {
	open: boolean;
	parseResult: ParseSopExcelResult | null;
	onClose: () => void;
	onRegistered: () => void;
};

const columns: ColumnsType<BulkRowWithResult<ParsedSopRow>> = [
	{
		title: 'SOP ID',
		key: 'sopId',
		render: (_, record): string =>
			record.document?.sopId ?? record.raw.sop_id ?? '',
	},
	{
		title: 'Title',
		key: 'title',
		render: (_, record): string =>
			record.document?.title ?? record.raw.title ?? '',
	},
	{
		title: 'Version',
		key: 'version',
		width: 120,
		render: (_, record): string =>
			record.document?.version ?? record.raw.version ?? '',
	},
	{
		title: 'Owner',
		key: 'owner',
		width: 110,
		render: (_, record): string =>
			record.document?.ownerTeam ?? record.raw.owner_team ?? '',
	},
];

function SopBulkPreviewDrawer({
	open,
	parseResult,
	onClose,
	onRegistered,
}: Props): JSX.Element {
	// SOP는 batch 엔드포인트가 있어 1회 호출로 끝난다. 응답 results는 요청에 실은
	// 유효 행 순서와 1:1이라 인덱스로 rowIndex에 되짚는다.
	const handleRegister = useCallback(
		async (validRows: ParsedSopRow[]): Promise<Map<number, BulkRowResult>> => {
			const response = await createSopDocumentBatch({
				contractVersion: SOP_DOCUMENT_LIST_CONTRACT_VERSION,
				documents: validRows.map((r) => r.document!),
			});
			const resultMap = new Map<number, BulkRowResult>();
			response.data.results.forEach((res, idx) => {
				if (validRows[idx]) {
					resultMap.set(validRows[idx].rowIndex, {
						status: res.status,
						error: res.error,
					});
				}
			});
			return resultMap;
		},
		[],
	);

	return (
		<BulkPreviewDrawer<ParsedSopRow>
			columns={columns}
			errorRowClassName="sop-preview-row--error"
			onClose={onClose}
			onRegister={handleRegister}
			onRegistered={onRegistered}
			open={open}
			parseResult={parseResult}
			statusColumnTitle="상태"
			statusColumnWidth={120}
			title="SOP 일괄 업로드 미리보기"
		/>
	);
}

export default SopBulkPreviewDrawer;
