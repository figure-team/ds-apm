import { useCallback, useState } from 'react';
import { Alert, Button, Drawer, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
	createSopDocumentBatch,
	SOP_DOCUMENT_LIST_CONTRACT_VERSION,
	type SopDocumentBatchResult,
} from 'api/v2/rules/sopDocuments';
import type { ParseSopExcelResult, ParsedSopRow } from './parseSopExcel';

type Props = {
	open: boolean;
	parseResult: ParseSopExcelResult | null;
	onClose: () => void;
	onRegistered: () => void;
};

type RowWithResult = ParsedSopRow & { batchResult?: SopDocumentBatchResult };

function getErrorMessage(error: unknown): string {
	if (typeof error === 'object' && error !== null && 'response' in error) {
		const response = (
			error as { response?: { data?: { error?: string; message?: string } | string } }
		).response;
		if (typeof response?.data === 'string') return response.data;
		return response?.data?.error || response?.data?.message || '요청 실패';
	}
	return error instanceof Error ? error.message : '요청 실패';
}

function SopBulkPreviewDrawer({
	open,
	parseResult,
	onClose,
	onRegistered,
}: Props): JSX.Element {
	const [submitting, setSubmitting] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [rowResults, setRowResults] = useState<Map<number, SopDocumentBatchResult>>(
		new Map(),
	);

	const rows: RowWithResult[] = (parseResult?.rows ?? []).map((row) => ({
		...row,
		batchResult: rowResults.get(row.rowIndex),
	}));

	const validDocs = rows.filter((r) => r.valid).map((r) => r.document!);

	const handleRegister = useCallback(async (): Promise<void> => {
		setSubmitting(true);
		setSubmitError('');
		try {
			const response = await createSopDocumentBatch({
				contractVersion: SOP_DOCUMENT_LIST_CONTRACT_VERSION,
				documents: validDocs,
			});
			const resultMap = new Map<number, SopDocumentBatchResult>();
			const validRows = rows.filter((r) => r.valid);
			response.data.results.forEach((res, idx) => {
				if (validRows[idx]) {
					resultMap.set(validRows[idx].rowIndex, res);
				}
			});
			setRowResults(resultMap);
			onRegistered();
		} catch (err) {
			setSubmitError(getErrorMessage(err));
		} finally {
			setSubmitting(false);
		}
	}, [validDocs, rows, onRegistered]);

	const handleClose = (): void => {
		setSubmitError('');
		setRowResults(new Map());
		onClose();
	};

	const columns: ColumnsType<RowWithResult> = [
		{
			title: '행',
			dataIndex: 'rowIndex',
			key: 'rowIndex',
			width: 50,
		},
		{
			title: 'SOP ID',
			key: 'sopId',
			render: (_, record): string => record.document?.sopId ?? record.raw.sop_id ?? '',
		},
		{
			title: 'Title',
			key: 'title',
			render: (_, record): string => record.document?.title ?? record.raw.title ?? '',
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
		{
			title: '상태',
			key: 'status',
			width: 120,
			render: (_, record): JSX.Element => {
				if (record.batchResult) {
					return record.batchResult.status === 'ok' ? (
						<Tag color="green">등록 완료</Tag>
					) : (
						<Tag color="red">서버 오류</Tag>
					);
				}
				return record.valid ? (
					<Tag color="blue">유효</Tag>
				) : (
					<Tag color="red">오류</Tag>
				);
			},
		},
		{
			title: '오류 내용',
			key: 'error',
			render: (_, record): string | undefined =>
				record.batchResult?.error ?? record.error,
		},
	];

	const registeredCount = [...rowResults.values()].filter(
		(r) => r.status === 'ok',
	).length;
	const serverErrorCount = [...rowResults.values()].filter(
		(r) => r.status === 'error',
	).length;

	return (
		<Drawer
			extra={
				rowResults.size === 0 ? (
					<Button
						disabled={validDocs.length === 0}
						loading={submitting}
						onClick={handleRegister}
						type="primary"
					>
						{validDocs.length}건 일괄 등록
					</Button>
				) : null
			}
			onClose={handleClose}
			open={open}
			size="large"
			title="SOP 일괄 업로드 미리보기"
			width={900}
		>
			<div style={{ marginBottom: 12 }}>
				{rowResults.size === 0 ? (
					<Typography.Text type="secondary">
						유효 {parseResult?.validCount ?? 0}건 / 오류{' '}
						{parseResult?.errorCount ?? 0}건
					</Typography.Text>
				) : (
					<Typography.Text>
						등록 완료 {registeredCount}건 / 서버 오류 {serverErrorCount}건
					</Typography.Text>
				)}
			</div>
			{submitError && (
				<Alert
					message={submitError}
					showIcon
					style={{ marginBottom: 12 }}
					type="error"
				/>
			)}
			<Table
				columns={columns}
				dataSource={rows}
				pagination={false}
				rowKey="rowIndex"
				rowClassName={(record): string =>
					!record.valid ||
					(record.batchResult && record.batchResult.status === 'error')
						? 'sop-preview-row--error'
						: ''
				}
				scroll={{ x: 800 }}
				size="small"
			/>
		</Drawer>
	);
}

export default SopBulkPreviewDrawer;
