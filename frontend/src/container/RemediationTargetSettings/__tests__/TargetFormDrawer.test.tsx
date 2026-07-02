import { fireEvent, render, screen, waitFor } from 'tests/test-utils';
import {
	createRemediationTarget,
	fetchHostKeyFingerprint,
	generateRemediationKeyPair,
	RemediationTargetWire,
	testRemediationConnection,
	updateRemediationTarget,
} from 'api/remediationTargets';

import TargetFormDrawer, { TargetFormDrawerProps } from '../TargetFormDrawer';

jest.mock('api/remediationTargets', () => ({
	createRemediationTarget: jest.fn(),
	updateRemediationTarget: jest.fn(),
	generateRemediationKeyPair: jest.fn(),
	fetchHostKeyFingerprint: jest.fn(),
	testRemediationConnection: jest.fn(),
}));

// Service options are a best-effort convenience for the tags Select; stub the
// hook so the drawer never fires a real /services request from the test env.
jest.mock('hooks/useQueryService', () => ({
	useQueryService: (): { data: unknown[]; isLoading: boolean; error: null } => ({
		data: [],
		isLoading: false,
		error: null,
	}),
}));

const mockCreate = createRemediationTarget as jest.MockedFunction<
	typeof createRemediationTarget
>;
const mockUpdate = updateRemediationTarget as jest.MockedFunction<
	typeof updateRemediationTarget
>;
const mockKeygen = generateRemediationKeyPair as jest.MockedFunction<
	typeof generateRemediationKeyPair
>;
const mockFingerprint = fetchHostKeyFingerprint as jest.MockedFunction<
	typeof fetchHostKeyFingerprint
>;
const mockTest = testRemediationConnection as jest.MockedFunction<
	typeof testRemediationConnection
>;

const PUBLIC_KEY = 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAKEYDATA dsapm-remediation';
const SEALED_BLOB = 'SEALED-PRIVATE-KEY-BLOB';
const FINGERPRINT = 'SHA256:abc123def456';

function baseProps(
	overrides?: Partial<TargetFormDrawerProps>,
): TargetFormDrawerProps {
	return {
		open: true,
		mode: 'create',
		encryptionReady: true,
		onClose: jest.fn(),
		onSaved: jest.fn(),
		...overrides,
	};
}

const editInitial: RemediationTargetWire = {
	id: '11111111-1111-4111-8111-111111111111',
	orgId: 'org-1',
	name: 'prod-web-01',
	host: '10.0.0.5',
	port: 22,
	user: 'deploy',
	credentialKind: 'private_key',
	hostKeyFingerprint: FINGERPRINT,
	serviceSelectors: ['payments-api'],
	hasCredential: true,
	createdAt: '2026-07-01T00:00:00Z',
	updatedAt: '2026-07-01T00:00:00Z',
};

// antd Select (mode="tags"): open, type, Enter creates the tag.
function addServiceTag(value: string): void {
	const combo = screen.getByRole('combobox');
	fireEvent.mouseDown(combo);
	fireEvent.change(combo, { target: { value } });
	fireEvent.keyDown(combo, { key: 'Enter', code: 'Enter', keyCode: 13 });
}

function fillDraftIdentity(): void {
	fireEvent.change(screen.getByTestId('target-name'), {
		target: { value: 'prod-web-01' },
	});
	fireEvent.change(screen.getByTestId('target-host'), {
		target: { value: '10.0.0.5' },
	});
	fireEvent.change(screen.getByTestId('target-user'), {
		target: { value: 'deploy' },
	});
}

describe('TargetFormDrawer', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockFingerprint.mockResolvedValue({
			fingerprint: FINGERPRINT,
			keyType: 'ssh-ed25519',
		});
		mockKeygen.mockResolvedValue({
			publicKeyOpenSSH: PUBLIC_KEY,
			sealedPrivateKey: SEALED_BLOB,
		});
		mockCreate.mockResolvedValue(editInitial);
		mockUpdate.mockResolvedValue(editInitial);
		mockTest.mockResolvedValue({ ok: true, exitCode: 0, output: 'ok' });
	});

	it('keygen 흐름: 공개키 표시 후 저장 시 sealedPrivateKey를 담아 생성한다 (평문 키 없음)', async () => {
		render(<TargetFormDrawer {...baseProps()} />);

		fillDraftIdentity();
		addServiceTag('payments-api');

		fireEvent.click(screen.getByTestId('fetch-fingerprint-btn'));
		await waitFor(() =>
			expect(screen.getByTestId('target-fingerprint')).toHaveValue(FINGERPRINT),
		);

		fireEvent.click(screen.getByTestId('keygen-btn'));
		await waitFor(() =>
			expect(screen.getByDisplayValue(PUBLIC_KEY)).toBeInTheDocument(),
		);

		fireEvent.click(screen.getByTestId('save-btn'));

		await waitFor(() => expect(mockCreate).toHaveBeenCalled());
		const body = mockCreate.mock.calls[0][0];
		expect(body.credential).toEqual({
			kind: 'private_key',
			sealedPrivateKey: SEALED_BLOB,
		});
		expect(body.credential?.privateKeyPEM).toBeUndefined();
		expect(body).toMatchObject({
			name: 'prod-web-01',
			host: '10.0.0.5',
			user: 'deploy',
			port: 22,
			hostKeyFingerprint: FINGERPRINT,
			serviceSelectors: ['payments-api'],
		});
	});

	it('지문 가져오기: 클릭 시 지문 필드가 채워진다', async () => {
		render(<TargetFormDrawer {...baseProps()} />);

		fireEvent.change(screen.getByTestId('target-host'), {
			target: { value: '10.0.0.5' },
		});
		fireEvent.click(screen.getByTestId('fetch-fingerprint-btn'));

		await waitFor(() =>
			expect(screen.getByTestId('target-fingerprint')).toHaveValue(FINGERPRINT),
		);
		expect(mockFingerprint).toHaveBeenCalledWith('10.0.0.5', 22);
	});

	it('host 변경 시 지문을 클리어하고 연결 테스트를 비활성화한다', async () => {
		render(<TargetFormDrawer {...baseProps()} />);

		fireEvent.change(screen.getByTestId('target-host'), {
			target: { value: '10.0.0.5' },
		});
		fireEvent.click(screen.getByTestId('fetch-fingerprint-btn'));
		await waitFor(() =>
			expect(screen.getByTestId('target-fingerprint')).toHaveValue(FINGERPRINT),
		);

		fireEvent.change(screen.getByTestId('target-host'), {
			target: { value: '10.0.0.9' },
		});

		expect(screen.getByTestId('target-fingerprint')).toHaveValue('');
		expect(screen.getByTestId('test-connection-btn')).toBeDisabled();
	});

	it('지문 미수집 시 연결 테스트 버튼이 비활성이다', () => {
		render(<TargetFormDrawer {...baseProps()} />);
		expect(screen.getByTestId('test-connection-btn')).toBeDisabled();
	});

	it('연결 테스트 성공/실패 결과를 인라인으로 표시한다', async () => {
		const { rerender } = render(<TargetFormDrawer {...baseProps()} />);

		fireEvent.change(screen.getByTestId('target-host'), {
			target: { value: '10.0.0.5' },
		});
		fireEvent.change(screen.getByTestId('target-user'), {
			target: { value: 'deploy' },
		});
		fireEvent.click(screen.getByTestId('fetch-fingerprint-btn'));
		await waitFor(() =>
			expect(screen.getByTestId('target-fingerprint')).toHaveValue(FINGERPRINT),
		);
		fireEvent.click(screen.getByTestId('keygen-btn'));
		await waitFor(() =>
			expect(screen.getByDisplayValue(PUBLIC_KEY)).toBeInTheDocument(),
		);

		fireEvent.click(screen.getByTestId('test-connection-btn'));
		await waitFor(() =>
			expect(screen.getByTestId('test-result-success')).toBeInTheDocument(),
		);

		mockTest.mockResolvedValueOnce({
			ok: false,
			exitCode: 1,
			error: 'handshake failed',
		});
		rerender(<TargetFormDrawer {...baseProps()} />);
		fireEvent.change(screen.getByTestId('target-host'), {
			target: { value: '10.0.0.5' },
		});
		fireEvent.click(screen.getByTestId('fetch-fingerprint-btn'));
		await waitFor(() =>
			expect(screen.getByTestId('target-fingerprint')).toHaveValue(FINGERPRINT),
		);
		fireEvent.click(screen.getByTestId('keygen-btn'));
		await waitFor(() =>
			expect(screen.getByDisplayValue(PUBLIC_KEY)).toBeInTheDocument(),
		);
		fireEvent.click(screen.getByTestId('test-connection-btn'));
		await waitFor(() =>
			expect(screen.getByTestId('test-result-error')).toBeInTheDocument(),
		);
	});

	it('수정 모드 "기존 키 유지": 저장 body에 credential이 없다', async () => {
		render(
			<TargetFormDrawer
				{...baseProps({ mode: 'edit', initial: editInitial })}
			/>,
		);

		fireEvent.click(screen.getByTestId('save-btn'));

		await waitFor(() => expect(mockUpdate).toHaveBeenCalled());
		const [id, body] = mockUpdate.mock.calls[0];
		expect(id).toBe(editInitial.id);
		expect(body.credential).toBeUndefined();
	});

	it('encryptionReady=false면 키 생성 버튼이 비활성이다', () => {
		render(<TargetFormDrawer {...baseProps({ encryptionReady: false })} />);
		expect(screen.getByTestId('keygen-btn')).toBeDisabled();
	});
});
