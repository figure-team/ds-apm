import { act, fireEvent, render, screen, waitFor, within } from 'tests/test-utils';

import type { TargetHealthWire } from 'api/remediationTargets';

import RemediationTargetSettings from '../RemediationTargetSettings';

// ── API mocks ──────────────────────────────────────────────────────────────
const mockList = jest.fn();
const mockDelete = jest.fn();
const mockTest = jest.fn();

jest.mock('api/remediationTargets', () => ({
	__esModule: true,
	listRemediationTargets: (...args: unknown[]): unknown => mockList(...args),
	deleteRemediationTarget: (...args: unknown[]): unknown => mockDelete(...args),
	testRemediationConnection: (...args: unknown[]): unknown => mockTest(...args),
}));

// ── Fixtures ───────────────────────────────────────────────────────────────
const target1 = {
	id: 'tgt-1',
	orgId: 'org-1',
	name: 'web-01',
	host: '10.0.0.1',
	port: 22,
	user: 'ubuntu',
	credentialKind: 'private_key',
	hostKeyFingerprint: 'SHA256:abcdefghijklmnopqrstuvwxyz012345',
	serviceSelectors: ['payment-api'],
	hasCredential: true,
	createdAt: '2026-07-01T00:00:00Z',
	updatedAt: '2026-07-01T00:00:00Z',
};

const target2 = {
	...target1,
	id: 'tgt-2',
	name: 'db-01',
	host: '10.0.0.2',
	port: 2222,
	user: 'postgres',
	serviceSelectors: ['billing-db', 'ledger'],
};

function mockListResponse(
	targets: Array<typeof target1 & { health?: TargetHealthWire }>,
	encryptionReady = true,
): void {
	mockList.mockResolvedValue({ targets, encryptionReady });
}

beforeEach(() => {
	jest.clearAllMocks();
	mockListResponse([target1, target2]);
	mockDelete.mockResolvedValue(undefined);
	mockTest.mockResolvedValue({ ok: true, exitCode: 0, output: 'ok' });
});

describe('RemediationTargetSettings', () => {
	// 목록 렌더: 타겟 2건 → 이름·host:port 셀 표시
	it('renders target rows with name and host:port', async () => {
		render(<RemediationTargetSettings />);

		expect(await screen.findByText('web-01')).toBeInTheDocument();
		expect(screen.getByText('db-01')).toBeInTheDocument();
		expect(screen.getByText('10.0.0.1:22')).toBeInTheDocument();
		expect(screen.getByText('10.0.0.2:2222')).toBeInTheDocument();
		expect(mockList).toHaveBeenCalledTimes(1);
	});

	// 빈 상태: 0건 → Empty 안내 문구
	it('renders empty guidance when there are no targets', async () => {
		mockListResponse([]);
		render(<RemediationTargetSettings />);

		expect(
			await screen.findByText('등록된 타겟이 없습니다'),
		).toBeInTheDocument();
	});

	// encryptionReady=false → 배너 + 추가 버튼 disabled
	it('shows the master key banner and disables add when encryption is not ready', async () => {
		mockListResponse([target1], false);
		render(<RemediationTargetSettings />);

		expect(
			await screen.findByText(
				'암호화 마스터키가 설정되지 않아 원격 타겟을 등록할 수 없습니다 (DS_APM_AI_CONFIG_ENCRYPTION_KEY)',
			),
		).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '타겟 추가' })).toBeDisabled();
	});

	// encryptionReady=true → 배너 없음 + 추가 버튼 enabled
	it('hides the banner and enables add when encryption is ready', async () => {
		render(<RemediationTargetSettings />);

		await screen.findByText('web-01');
		expect(
			screen.queryByText(
				'암호화 마스터키가 설정되지 않아 원격 타겟을 등록할 수 없습니다 (DS_APM_AI_CONFIG_ENCRYPTION_KEY)',
			),
		).not.toBeInTheDocument();
		expect(screen.getByRole('button', { name: '타겟 추가' })).toBeEnabled();
	});

	// 삭제: 삭제 버튼 → 확인 모달 → confirm 시 deleteRemediationTarget 호출 + refetch
	it('confirms deletion via modal and calls deleteRemediationTarget', async () => {
		render(<RemediationTargetSettings />);

		await screen.findByText('web-01');
		const firstRow = screen.getByText('web-01').closest('tr') as HTMLElement;
		fireEvent.click(within(firstRow).getByRole('button', { name: '삭제' }));

		const dialog = await screen.findByRole('dialog');
		expect(within(dialog).getByText(/web-01/)).toBeInTheDocument();
		fireEvent.click(within(dialog).getByRole('button', { name: '삭제' }));

		await waitFor(() => {
			expect(mockDelete).toHaveBeenCalledWith('tgt-1');
		});
		// 삭제 성공 후 목록 refetch
		await waitFor(() => {
			expect(mockList).toHaveBeenCalledTimes(2);
		});
	});

	// 행 테스트: testRemediationConnection이 {targetId}로 호출되고 성공 배지 표시
	it('runs a row connection test and shows a success badge', async () => {
		render(<RemediationTargetSettings />);

		await screen.findByText('web-01');
		const firstRow = screen.getByText('web-01').closest('tr') as HTMLElement;
		fireEvent.click(within(firstRow).getByRole('button', { name: '테스트' }));

		await waitFor(() => {
			expect(mockTest).toHaveBeenCalledWith({ targetId: 'tgt-1' });
		});
		expect(await within(firstRow).findByText('성공')).toBeInTheDocument();
	});

	// 행 테스트 실패: 실패 배지 표시
	it('shows a failure badge when the row connection test fails', async () => {
		mockTest.mockResolvedValue({
			ok: false,
			exitCode: 1,
			error: 'handshake failed',
		});
		render(<RemediationTargetSettings />);

		await screen.findByText('web-01');
		const firstRow = screen.getByText('web-01').closest('tr') as HTMLElement;
		fireEvent.click(within(firstRow).getByRole('button', { name: '테스트' }));

		expect(await within(firstRow).findByText('실패')).toBeInTheDocument();
	});

	// 헬스 배지: 4상태 렌더 (healthy/unreachable/mismatch/부재→확인 중)
	it('renders a health badge per target state', async () => {
		mockListResponse([
			{
				...target1,
				health: { status: 'healthy', checkedAt: '2026-07-10T02:00:00Z' },
			},
			{
				...target2,
				health: {
					status: 'unreachable',
					checkedAt: '2026-07-10T02:00:00Z',
					error: 'dial tcp 10.0.0.2:2222: i/o timeout',
				},
			},
			{
				...target1,
				id: 'tgt-3',
				name: 'cache-01',
				health: { status: 'mismatch', checkedAt: '2026-07-10T02:00:00Z' },
			},
			{ ...target1, id: 'tgt-4', name: 'batch-01' },
		]);
		render(<RemediationTargetSettings />);

		expect(await screen.findByText('정상')).toBeInTheDocument();
		expect(screen.getByText('연결 불가')).toBeInTheDocument();
		expect(screen.getByText('호스트키 불일치')).toBeInTheDocument();
		expect(screen.getByText('확인 중')).toBeInTheDocument();
	});

	// 60초 인터벌 재조회 — 스피너 없이(silent) 목록만 갱신
	it('silently refreshes the list on the 60s interval', async () => {
		const setIntervalSpy = jest.spyOn(window, 'setInterval');
		render(<RemediationTargetSettings />);
		expect(await screen.findByText('web-01')).toBeInTheDocument();
		expect(mockList).toHaveBeenCalledTimes(1);

		const call = setIntervalSpy.mock.calls.find(([, ms]) => ms === 60000);
		expect(call).toBeDefined();
		const tick = call?.[0] as () => void;

		act(() => {
			tick();
		});
		await waitFor(() => expect(mockList).toHaveBeenCalledTimes(2));
		// silent: 재조회 중에도 테이블 스피너가 돌지 않는다
		expect(document.querySelector('.ant-spin-spinning')).toBeNull();
		setIntervalSpy.mockRestore();
	});
});
