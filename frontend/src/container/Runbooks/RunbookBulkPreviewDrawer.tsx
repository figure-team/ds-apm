import { useCallback, useState } from 'react';
import { Button, Drawer, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import createRunbook from 'api/runbook/createRunbook';
import { useIsDarkMode } from 'hooks/useDarkMode';

import type {
	ParsedRunbookRow,
	ParseRunbookExcelResult,
} from './parseRunbookExcel';
import './Runbooks.styles.scss';

type RowResult = { status: 'ok' | 'error'; error?: string };

type RowWithResult = ParsedRunbookRow & { result?: RowResult };

type Props = {
	open: boolean;
	parseResult: ParseRunbookExcelResult | null;
	sopId: string;
	version: string;
	onClose: () => void;
	onRegistered: () => void;
};

function getErrorMessage(error: unknown): string {
	if (typeof error === 'object' && error !== null && 'response' in error) {
		const response = (
			error as {
				response?: { data?: { error?: string; message?: string } | string };
			}
		).response;
		if (typeof response?.data === 'string') {
			return response.data;
		}
		return response?.data?.error || response?.data?.message || '요청 실패';
	}
	return error instanceof Error ? error.message : '요청 실패';
}

function RunbookBulkPreviewDrawer({
	open,
	parseResult,
	sopId,
	version,
	onClose,
	onRegistered,
}: Props): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const [submitting, setSubmitting] = useState(false);
	const [rowResults, setRowResults] = useState<Map<number, RowResult>>(
		new Map(),
	);

	const rows: RowWithResult[] = (parseResult?.rows ?? []).map((row) => ({
		...row,
		result: rowResults.get(row.rowIndex),
	}));

	const validRows = rows.filter((r) => r.valid);

	// Runbook은 batch 엔드포인트가 없어 행별로 순차 생성한다. 한 건이 실패해도
	// 나머지는 계속 진행하고, 각 행의 성공/실패를 표에 반영한다.
	const handleRegister = useCallback(async (): Promise<void> => {
		setSubmitting(true);
		const results = new Map<number, RowResult>();
		for (const row of validRows) {
			try {
				await createRunbook(sopId, version, row.runbook ?? {});
				results.set(row.rowIndex, { status: 'ok' });
			} catch (err) {
				results.set(row.rowIndex, {
					status: 'error',
					error: getErrorMessage(err),
				});
			}
		}
		setRowResults(results);
		setSubmitting(false);
		onRegistered();
	}, [validRows, sopId, version, onRegistered]);

	const handleClose = (): void => {
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
		{
			title: '결과',
			key: 'result',
			width: 110,
			render: (_, record): JSX.Element => {
				if (record.result) {
					return record.result.status === 'ok' ? (
						<Tag color={isDarkMode ? 'green' : '#16A34A'}>등록 완료</Tag>
					) : (
						<Tag color={isDarkMode ? 'red' : '#DC2626'}>서버 오류</Tag>
					);
				}
				return record.valid ? (
					<Tag color={isDarkMode ? 'blue' : '#2563EB'}>유효</Tag>
				) : (
					<Tag color={isDarkMode ? 'red' : '#DC2626'}>오류</Tag>
				);
			},
		},
		{
			title: '오류 내용',
			key: 'error',
			render: (_, record): string | undefined =>
				record.result?.error ?? record.error,
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
						disabled={validRows.length === 0}
						loading={submitting}
						onClick={handleRegister}
						type="primary"
					>
						{validRows.length}건 일괄 등록
					</Button>
				) : null
			}
			onClose={handleClose}
			open={open}
			size="large"
			title="Runbook 일괄 업로드 미리보기"
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
			<Table
				columns={columns}
				dataSource={rows}
				pagination={false}
				rowKey="rowIndex"
				rowClassName={(record): string =>
					!record.valid || record.result?.status === 'error'
						? 'runbook-preview-row--error'
						: ''
				}
				scroll={{ x: 800 }}
				size="small"
			/>
		</Drawer>
	);
}

export default RunbookBulkPreviewDrawer;
