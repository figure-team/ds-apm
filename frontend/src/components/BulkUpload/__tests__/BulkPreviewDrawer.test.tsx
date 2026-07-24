import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import BulkPreviewDrawer from '../BulkPreviewDrawer';
import type { BulkParseResult, BulkRowBase, BulkRowResult } from '../types';

jest.mock('hooks/useDarkMode', () => ({
	useIsDarkMode: (): boolean => false,
}));

type Row = BulkRowBase & { payload?: { name: string } };

const PARSE_RESULT: BulkParseResult<Row> = {
	rows: [
		{ rowIndex: 2, valid: true, raw: { name: 'a' }, payload: { name: 'a' } },
		{ rowIndex: 3, valid: false, error: '필수 필드 누락: name', raw: {} },
	],
	validCount: 1,
	errorCount: 1,
};

const COLUMNS = [
	{
		title: '이름',
		key: 'name',
		render: (_: unknown, record: Row): string => record.payload?.name ?? '',
	},
];

function renderDrawer(
	overrides: Partial<React.ComponentProps<typeof BulkPreviewDrawer<Row>>> = {},
): {
	onRegister: jest.Mock;
	onRegistered: jest.Mock;
	onClose: jest.Mock;
} {
	const onRegister = jest.fn(async () => new Map<number, BulkRowResult>());
	const onRegistered = jest.fn();
	const onClose = jest.fn();
	render(
		<BulkPreviewDrawer<Row>
			columns={COLUMNS}
			errorRowClassName="demo-row--error"
			onClose={onClose}
			onRegister={onRegister}
			onRegistered={onRegistered}
			open
			parseResult={PARSE_RESULT}
			statusColumnTitle="상태"
			statusColumnWidth={120}
			title="일괄 업로드 미리보기"
			{...overrides}
		/>,
	);
	return { onRegister, onRegistered, onClose };
}

describe('BulkPreviewDrawer', () => {
	it('컬럼을 [행, 도메인, 상태, 오류 내용] 순으로 조립한다', () => {
		renderDrawer();
		expect(
			screen.getAllByRole('columnheader').map((th) => th.textContent),
		).toStrictEqual(['행', '이름', '상태', '오류 내용']);
	});

	it('등록 전에는 파싱 요약을, 오류 행에는 오류 사유를 보여준다', () => {
		renderDrawer();
		expect(screen.getByText(/유효 1건/)).toBeInTheDocument();
		expect(screen.getByText('필수 필드 누락: name')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '1건 일괄 등록' })).toBeEnabled();
	});

	it('유효 행이 없으면 등록 버튼을 막는다', () => {
		renderDrawer({
			parseResult: { rows: [PARSE_RESULT.rows[1]], validCount: 0, errorCount: 1 },
		});
		expect(screen.getByRole('button', { name: '0건 일괄 등록' })).toBeDisabled();
	});

	it('유효 행만 등록에 넘기고 행별 결과를 표에 반영한다', async () => {
		const onRegister = jest.fn(
			async () => new Map<number, BulkRowResult>([[2, { status: 'ok' }]]),
		);
		const { onRegistered } = renderDrawer({ onRegister });

		fireEvent.click(screen.getByRole('button', { name: '1건 일괄 등록' }));

		await waitFor(() => expect(onRegistered).toHaveBeenCalledTimes(1));
		expect(onRegister).toHaveBeenCalledWith([
			expect.objectContaining({ rowIndex: 2 }),
		]);
		expect(screen.getByText('등록 완료')).toBeInTheDocument();
		expect(
			screen.getByText(/등록 완료 1건 \/ 서버 오류 0건/),
		).toBeInTheDocument();
		// 결과가 채워지면 재등록 버튼은 사라진다
		expect(screen.queryByRole('button', { name: /일괄 등록/ })).toBeNull();
	});

	it('등록 호출 자체가 실패하면 Alert로 알리고 완료 콜백을 부르지 않는다', async () => {
		const onRegister = jest.fn(async () => {
			throw { response: { data: { error: '권한이 없습니다' } } };
		});
		const { onRegistered } = renderDrawer({ onRegister });

		fireEvent.click(screen.getByRole('button', { name: '1건 일괄 등록' }));

		await waitFor(() =>
			expect(screen.getByText('권한이 없습니다')).toBeInTheDocument(),
		);
		expect(onRegistered).not.toHaveBeenCalled();
	});
});
