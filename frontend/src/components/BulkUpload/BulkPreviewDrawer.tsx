import { useCallback, useState } from 'react';
import { Alert, Button, Drawer, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useIsDarkMode } from 'hooks/useDarkMode';

import { getBulkErrorMessage } from './getBulkErrorMessage';
import type {
	BulkParseResult,
	BulkRowBase,
	BulkRowResult,
	BulkRowWithResult,
} from './types';

type Props<TRow extends BulkRowBase> = {
	open: boolean;
	title: string;
	parseResult: BulkParseResult<TRow> | null;
	/** 도메인 컬럼. 표는 [행, ...이 컬럼들, 상태 태그, 오류 내용] 순으로 조립된다. */
	columns: ColumnsType<BulkRowWithResult<TRow>>;
	statusColumnTitle: string;
	statusColumnWidth: number;
	/** 오류 행에 붙일 클래스. 도메인 scss에 정의돼 있다. */
	errorRowClassName: string;
	/**
	 * 유효 행들을 서버에 등록하고 rowIndex → 결과 맵을 돌려준다.
	 * batch 엔드포인트가 있으면 1회 호출, 없으면 행별 순차 호출 — 그 차이를 여기서 흡수한다.
	 * reject하면 표 대신 상단 Alert로 표시된다.
	 */
	onRegister: (validRows: TRow[]) => Promise<Map<number, BulkRowResult>>;
	onClose: () => void;
	onRegistered: () => void;
};

/** 파싱 결과를 표로 보여주고 유효 행만 일괄 등록하는 미리보기 드로어. */
function BulkPreviewDrawer<TRow extends BulkRowBase>({
	open,
	title,
	parseResult,
	columns,
	statusColumnTitle,
	statusColumnWidth,
	errorRowClassName,
	onRegister,
	onClose,
	onRegistered,
}: Props<TRow>): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const [submitting, setSubmitting] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [rowResults, setRowResults] = useState<Map<number, BulkRowResult>>(
		new Map(),
	);

	const rows: BulkRowWithResult<TRow>[] = (parseResult?.rows ?? []).map(
		(row) => ({ ...row, result: rowResults.get(row.rowIndex) }),
	);
	const validRows = rows.filter((r) => r.valid);

	const handleRegister = useCallback(async (): Promise<void> => {
		setSubmitting(true);
		setSubmitError('');
		try {
			setRowResults(await onRegister(validRows));
			onRegistered();
		} catch (err) {
			setSubmitError(getBulkErrorMessage(err));
		} finally {
			setSubmitting(false);
		}
	}, [validRows, onRegister, onRegistered]);

	const handleClose = (): void => {
		setSubmitError('');
		setRowResults(new Map());
		onClose();
	};

	const allColumns: ColumnsType<BulkRowWithResult<TRow>> = [
		{ title: '행', dataIndex: 'rowIndex', key: 'rowIndex', width: 50 },
		...columns,
		{
			title: statusColumnTitle,
			key: 'bulkStatus',
			width: statusColumnWidth,
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
			key: 'bulkError',
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
			title={title}
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
				columns={allColumns}
				dataSource={rows}
				pagination={false}
				rowClassName={(record): string =>
					!record.valid || record.result?.status === 'error' ? errorRowClassName : ''
				}
				rowKey="rowIndex"
				scroll={{ x: 800 }}
				size="small"
			/>
		</Drawer>
	);
}

export default BulkPreviewDrawer;
