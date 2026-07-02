import './TargetFormDrawer.styles.scss';

import { CopyOutlined } from '@ant-design/icons';
import {
	Alert,
	Button,
	Drawer,
	Form,
	Input,
	InputNumber,
	Radio,
	Select,
	Space,
	Tabs,
	Tooltip,
	Typography,
} from 'antd';
import {
	createRemediationTarget,
	ConnectionTestRequest,
	fetchHostKeyFingerprint,
	generateRemediationKeyPair,
	RemediationTargetCredential,
	RemediationTargetUpsert,
	RemediationTargetWire,
	testRemediationConnection,
	ConnectionTestResult,
	updateRemediationTarget,
} from 'api/remediationTargets';
import { useQueryService } from 'hooks/useQueryService';
import { useNotifications } from 'hooks/useNotifications';
import { useCopyToClipboard } from 'hooks/useCopyToClipboard';
import { useMemo, useState } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { AppState } from 'store/reducers';
import { GlobalReducer } from 'types/reducer/globalTime';

// props 계약은 레인 간 인터페이스로 시드에 고정 — 변경은 오케스트레이터 승인 필요
// (plans/2026-07-02-remtgt-parallel-orchestration.md). 구현은 레인 C2 담당.
export interface TargetFormDrawerProps {
	open: boolean;
	mode: 'create' | 'edit';
	initial?: RemediationTargetWire;
	encryptionReady: boolean;
	onClose: () => void;
	onSaved: () => void;
}

type CredentialTab = 'generate' | 'paste';

interface TargetFormValues {
	name: string;
	host: string;
	port: number;
	user: string;
	serviceSelectors: string[];
}

const DEFAULT_PORT = 22;

function errorMessage(err: unknown): string {
	const maybe = err as
		| { response?: { data?: { error?: { message?: string } } }; message?: string }
		| undefined;
	return (
		maybe?.response?.data?.error?.message ||
		maybe?.message ||
		'요청을 처리하지 못했습니다'
	);
}

function TargetFormDrawer({
	open,
	mode,
	initial,
	encryptionReady,
	onClose,
	onSaved,
}: TargetFormDrawerProps): JSX.Element {
	const [form] = Form.useForm<TargetFormValues>();
	const { notifications } = useNotifications();
	const { copyToClipboard, isCopied } = useCopyToClipboard();

	// Host key fingerprint lives outside the antd Form: it is populated by the
	// "fetch fingerprint" round-trip, not typed. Seeded from the target on edit.
	const [fingerprint, setFingerprint] = useState<string>(
		initial?.hostKeyFingerprint ?? '',
	);
	const [keyType, setKeyType] = useState<string>('');
	const [fingerprintError, setFingerprintError] = useState<string | null>(null);
	const [fetchingFingerprint, setFetchingFingerprint] = useState(false);

	const [activeTab, setActiveTab] = useState<CredentialTab>('generate');
	// sealedPrivateKey is the stateless keygen blob — kept in state only, never
	// rendered, sent back verbatim on save (design §3.2). Plaintext never here.
	const [sealedPrivateKey, setSealedPrivateKey] = useState<string>('');
	const [publicKey, setPublicKey] = useState<string>('');
	const [pemText, setPemText] = useState<string>('');
	const [keygenLoading, setKeygenLoading] = useState(false);
	// Edit defaults to keeping the stored key (design §4.3).
	const [keepExistingKey, setKeepExistingKey] = useState(mode === 'edit');

	const [testResult, setTestResult] = useState<ConnectionTestResult | null>(null);
	const [testing, setTesting] = useState(false);
	const [submitting, setSubmitting] = useState(false);

	const hostValue = Form.useWatch('host', form);
	const portValue = Form.useWatch('port', form);

	// Service options are a best-effort convenience; the tags Select still accepts
	// free input for services with no traffic yet (design §4.3).
	const { maxTime, minTime, selectedTime } = useSelector<
		AppState,
		GlobalReducer
	>((state) => state.globalTime);
	const { data: services } = useQueryService({
		minTime,
		maxTime,
		selectedTime,
		selectedTags: [],
	});
	const serviceOptions = useMemo(
		() =>
			(services ?? []).map((svc) => ({
				label: svc.serviceName,
				value: svc.serviceName,
			})),
		[services],
	);

	const usingExistingKey = mode === 'edit' && keepExistingKey;

	const buildCredential = (): RemediationTargetCredential | undefined => {
		if (usingExistingKey) return undefined;
		if (activeTab === 'generate' && sealedPrivateKey) {
			return { kind: 'private_key', sealedPrivateKey };
		}
		if (activeTab === 'paste' && pemText.trim()) {
			return { kind: 'private_key', privateKeyPEM: pemText };
		}
		return undefined;
	};

	const credential = buildCredential();
	const needsNewCredential = !usingExistingKey;
	const missingCredential = needsNewCredential && !credential;
	// New credentials require the master key; keeping the stored key does not.
	const blockedByEncryption = needsNewCredential && !encryptionReady;
	const saveDisabled =
		submitting || blockedByEncryption || missingCredential;

	const handleValuesChange = (changed: Partial<TargetFormValues>): void => {
		// A stale fingerprint against a new host/port would always fail execution
		// with a host key mismatch — clear it so the test button forces a refetch.
		if ('host' in changed || 'port' in changed) {
			setFingerprint('');
			setKeyType('');
			setFingerprintError(null);
			setTestResult(null);
		}
	};

	const handleFetchFingerprint = async (): Promise<void> => {
		const { host, port } = form.getFieldsValue(['host', 'port']);
		if (!host || !port) return;
		setFetchingFingerprint(true);
		setFingerprintError(null);
		try {
			const res = await fetchHostKeyFingerprint(host, port);
			setFingerprint(res.fingerprint);
			setKeyType(res.keyType);
		} catch (err) {
			setFingerprintError(errorMessage(err));
		} finally {
			setFetchingFingerprint(false);
		}
	};

	const handleKeygen = async (): Promise<void> => {
		setKeygenLoading(true);
		try {
			const res = await generateRemediationKeyPair();
			setPublicKey(res.publicKeyOpenSSH);
			setSealedPrivateKey(res.sealedPrivateKey);
		} catch (err) {
			notifications.error({ message: errorMessage(err) });
		} finally {
			setKeygenLoading(false);
		}
	};

	const handleTest = async (): Promise<void> => {
		setTesting(true);
		setTestResult(null);
		try {
			let request: ConnectionTestRequest;
			if (usingExistingKey && initial) {
				request = { targetId: initial.id };
			} else {
				const { host, port, user } = form.getFieldsValue([
					'host',
					'port',
					'user',
				]);
				request = {
					host,
					port,
					user,
					hostKeyFingerprint: fingerprint,
					credential: buildCredential(),
				};
			}
			const res = await testRemediationConnection(request);
			setTestResult(res);
		} catch (err) {
			setTestResult({ ok: false, error: errorMessage(err) });
		} finally {
			setTesting(false);
		}
	};

	const handleSave = async (values: TargetFormValues): Promise<void> => {
		const cred = buildCredential();
		if (needsNewCredential && !cred) return;
		const body: RemediationTargetUpsert = {
			name: values.name,
			host: values.host,
			port: values.port,
			user: values.user,
			serviceSelectors: values.serviceSelectors,
			hostKeyFingerprint: fingerprint,
		};
		if (cred) body.credential = cred;
		setSubmitting(true);
		try {
			if (mode === 'edit' && initial) {
				await updateRemediationTarget(initial.id, body);
			} else {
				await createRemediationTarget(body);
			}
			onSaved();
			onClose();
		} catch (err) {
			notifications.error({ message: errorMessage(err) });
		} finally {
			setSubmitting(false);
		}
	};

	const credentialTabs = [
		{
			key: 'generate',
			label: '키 생성',
			children: (
				<div className="target-form-drawer__keygen">
					<Button
						data-testid="keygen-btn"
						disabled={!encryptionReady}
						loading={keygenLoading}
						onClick={handleKeygen}
					>
						키 생성
					</Button>
					{!encryptionReady && (
						<Typography.Text type="warning">
							암호화 마스터키가 설정되지 않아 키를 생성할 수 없습니다
							(DS_APM_AI_CONFIG_ENCRYPTION_KEY)
						</Typography.Text>
					)}
					{publicKey && (
						<div className="target-form-drawer__public-key">
							<Typography.Text>
								이 공개키를 대상 서버의 ~/.ssh/authorized_keys에 추가하세요
							</Typography.Text>
							<Input.TextArea
								data-testid="public-key"
								value={publicKey}
								readOnly
								autoSize={{ minRows: 2, maxRows: 4 }}
							/>
							<Button
								data-testid="copy-public-key-btn"
								icon={<CopyOutlined />}
								onClick={(): void => copyToClipboard(publicKey)}
							>
								{isCopied ? '복사됨' : '공개키 복사'}
							</Button>
						</div>
					)}
				</div>
			),
		},
		{
			key: 'paste',
			label: 'PEM 붙여넣기',
			children: (
				<Input.TextArea
					data-testid="pem-textarea"
					value={pemText}
					onChange={(e): void => setPemText(e.target.value)}
					placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
					autoSize={{ minRows: 4, maxRows: 10 }}
				/>
			),
		},
	];

	return (
		<Drawer
			className="target-form-drawer"
			title={mode === 'edit' ? '자동대응 타겟 수정' : '자동대응 타겟 추가'}
			open={open}
			onClose={onClose}
			width={560}
			destroyOnClose
			footer={
				<div className="target-form-drawer__footer">
					{testResult &&
						(testResult.ok ? (
							<Alert
								data-testid="test-result-success"
								type="success"
								showIcon
								message="연결 성공 (echo ok)"
							/>
						) : (
							<Alert
								data-testid="test-result-error"
								type="error"
								showIcon
								message="연결 실패"
								description={testResult.error || testResult.output}
							/>
						))}
					{missingCredential && mode === 'create' && (
						<Typography.Text type="secondary">
							키를 생성하거나 PEM을 입력하세요
						</Typography.Text>
					)}
					<Space>
						<Button data-testid="cancel-btn" onClick={onClose}>
							취소
						</Button>
						<Tooltip title={!fingerprint ? '먼저 지문을 가져오세요' : ''}>
							<Button
								data-testid="test-connection-btn"
								disabled={!fingerprint || testing}
								loading={testing}
								onClick={handleTest}
							>
								연결 테스트
							</Button>
						</Tooltip>
						<Button
							type="primary"
							data-testid="save-btn"
							disabled={saveDisabled}
							loading={submitting}
							onClick={(): void => form.submit()}
						>
							저장
						</Button>
					</Space>
				</div>
			}
		>
			<Form<TargetFormValues>
				form={form}
				layout="vertical"
				requiredMark
				initialValues={{
					name: initial?.name ?? '',
					host: initial?.host ?? '',
					port: initial?.port ?? DEFAULT_PORT,
					user: initial?.user ?? '',
					serviceSelectors: initial?.serviceSelectors ?? [],
				}}
				onValuesChange={handleValuesChange}
				onFinish={handleSave}
			>
				<Form.Item
					label="이름"
					name="name"
					rules={[{ required: true, message: '이름을 입력하세요' }]}
				>
					<Input data-testid="target-name" placeholder="prod-web-01" />
				</Form.Item>
				<Form.Item
					label="Host"
					name="host"
					rules={[{ required: true, message: 'Host를 입력하세요' }]}
				>
					<Input
						data-testid="target-host"
						placeholder="10.0.0.5 또는 host.example.com"
					/>
				</Form.Item>
				<Form.Item
					label="Port"
					name="port"
					rules={[{ required: true, message: 'Port를 입력하세요' }]}
				>
					<InputNumber
						data-testid="target-port"
						min={1}
						max={65535}
						style={{ width: '100%' }}
					/>
				</Form.Item>
				<Form.Item
					label="User"
					name="user"
					rules={[{ required: true, message: 'User를 입력하세요' }]}
				>
					<Input data-testid="target-user" placeholder="deploy" />
				</Form.Item>
				<Form.Item
					label="서비스 셀렉터"
					name="serviceSelectors"
					rules={[
						{
							required: true,
							type: 'array',
							min: 1,
							message: '서비스 셀렉터를 하나 이상 지정하세요',
						},
					]}
				>
					<Select
						data-testid="target-services"
						mode="tags"
						placeholder="서비스명 선택 또는 입력"
						options={serviceOptions}
						tokenSeparators={[',']}
					/>
				</Form.Item>

				<Form.Item label="호스트키 지문">
					<Space.Compact style={{ width: '100%' }}>
						<Input
							data-testid="target-fingerprint"
							value={fingerprint}
							readOnly
							placeholder="SHA256:..."
							suffix={keyType ? <span>{keyType}</span> : undefined}
						/>
						<Button
							data-testid="fetch-fingerprint-btn"
							disabled={!hostValue || !portValue || fetchingFingerprint}
							loading={fetchingFingerprint}
							onClick={handleFetchFingerprint}
						>
							지문 가져오기
						</Button>
					</Space.Compact>
					{fingerprintError && (
						<Typography.Text type="danger" data-testid="fingerprint-error">
							{fingerprintError}
						</Typography.Text>
					)}
				</Form.Item>

				<Form.Item label="자격증명">
					{mode === 'edit' && (
						<Radio.Group
							className="target-form-drawer__credential-mode"
							value={keepExistingKey ? 'keep' : 'replace'}
							onChange={(e): void =>
								setKeepExistingKey(e.target.value === 'keep')
							}
						>
							<Radio.Button value="keep" data-testid="keep-existing-key">
								기존 키 유지
							</Radio.Button>
							<Radio.Button value="replace" data-testid="replace-key">
								키 교체
							</Radio.Button>
						</Radio.Group>
					)}
					{!usingExistingKey && (
						<Tabs
							activeKey={activeTab}
							onChange={(key): void => setActiveTab(key as CredentialTab)}
							items={credentialTabs}
						/>
					)}
				</Form.Item>
			</Form>
		</Drawer>
	);
}

export default TargetFormDrawer;
