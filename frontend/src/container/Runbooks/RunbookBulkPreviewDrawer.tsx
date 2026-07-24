import { useCallback } from 'react';
import type { ColumnsType } from 'antd/es/table';
import createRunbook from 'api/runbook/createRunbook';
import BulkPreviewDrawer from 'components/BulkUpload/BulkPreviewDrawer';
import { getBulkErrorMessage } from 'components/BulkUpload/getBulkErrorMessage';
import type {
	BulkRowResult,
	BulkRowWithResult,
} from 'components/BulkUpload/types';

import type {
	ParsedRunbookRow,
	ParseRunbookExcelResult,
} from './parseRunbookExcel';
import './Runbooks.styles.scss';

type Props = {
	open: boolean;
	parseResult: ParseRunbookExcelResult | null;
	sopId: string;
	version: string;
	onClose: () => void;
	onRegistered: () => void;
};

const columns: ColumnsType<BulkRowWithResult<ParsedRunbookRow>> = [
	{
		title: '제목',
		key: 'title',
		render: (_, record): string =>
			record.runbook?.title ?? record.raw.title ?? '',
	},
	{
		title: '상태',
		key: 'runbookStatus',
		width: 100,
		render: (_, record): string =>
			record.runbook?.status ?? record.raw.status ?? 'draft',
	},
];

function RunbookBulkPreviewDrawer({
	open,
	parseResult,
	sopId,
	version,
	onClose,
	onRegistered,
}: Props): JSX.Element {
	// Runbook은 batch 엔드포인트가 없어 행별로 순차 생성한다. 한 건이 실패해도
	// 나머지는 계속 진행하고, 각 행의 성공/실패를 표에 반영한다.
	const handleRegister = useCallback(
		async (
			validRows: ParsedRunbookRow[],
		): Promise<Map<number, BulkRowResult>> => {
			const results = new Map<number, BulkRowResult>();
			for (const row of validRows) {
				try {
					await createRunbook(sopId, version, row.runbook ?? {});
					results.set(row.rowIndex, { status: 'ok' });
				} catch (err) {
					results.set(row.rowIndex, {
						status: 'error',
						error: getBulkErrorMessage(err),
					});
				}
			}
			return results;
		},
		[sopId, version],
	);

	return (
		<BulkPreviewDrawer<ParsedRunbookRow>
			columns={columns}
			errorRowClassName="runbook-preview-row--error"
			onClose={onClose}
			onRegister={handleRegister}
			onRegistered={onRegistered}
			open={open}
			parseResult={parseResult}
			statusColumnTitle="결과"
			statusColumnWidth={110}
			title="Runbook 일괄 업로드 미리보기"
		/>
	);
}

export default RunbookBulkPreviewDrawer;
