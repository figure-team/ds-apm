import { useCallback, useEffect, useRef, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { Badge, toast } from '@signozhq/ui';
import { Alert, Button, Input, Radio, Tag } from 'antd';
import getAIConfig from 'api/aiModule/getAIConfig';
import testAIConfig from 'api/aiModule/testAIConfig';
import updateAIConfig from 'api/aiModule/updateAIConfig';
import logEvent from 'api/common/logEvent';
import { AxiosError } from 'axios';
import Spinner from 'components/Spinner';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import './AIModuleSettings.styles.scss';

// ── Types ──────────────────────────────────────────────────────────────────

export interface AIConfig {
	contractVersion: string;
	orgId: string;
	provider: 'local' | 'mock' | 'llm';
	llmProvider: 'claude' | 'codex';
	transport: 'api' | 'cli';
	model: string;
	apiKey: string;
	oauthToken: string;
	binaryPath: string;
	timeoutSeconds: number;
	updatedAt: string;
}

export type AIConfigPayload = Pick<
	AIConfig,
	| 'provider'
	| 'llmProvider'
	| 'transport'
	| 'model'
	| 'apiKey'
	| 'oauthToken'
	| 'binaryPath'
	| 'timeoutSeconds'
>;

export type AIConfigErrorKind = 'auth' | 'timeout' | 'other';

export interface AIConfigTestResult {
	ok: boolean;
	headline?: string;
	model?: string;
	error?: string;
	errorKind?: AIConfigErrorKind;
}

interface FormValues {
	provider: 'local' | 'mock' | 'llm';
	llmProvider: 'claude' | 'codex';
	transport: 'api' | 'cli';
	model: string;
	apiKey: string;
	oauthToken: string;
	binaryPath: string;
	timeoutSeconds: number | '';
}

const API_KEY_UNCHANGED = '<unchanged>';

const ANALYTICS = {
	PAGE_VIEWED: 'AI Module Settings: Page viewed',
	SAVED: 'AI Module Settings: Configuration saved',
	TESTED: 'AI Module Settings: Test invoked',
} as const;

// ── Helpers ────────────────────────────────────────────────────────────────

function getErrorMessage(err: unknown, fallback: string): string {
	if (err instanceof Error) return err.message;
	const axErr = err as AxiosError<{ error?: string; message?: string }>;
	return (
		axErr?.response?.data?.error ?? axErr?.response?.data?.message ?? fallback
	);
}

function truncate(str: string, max: number): string {
	return str.length > max ? `${str.slice(0, max)}…` : str;
}

// ── Component ──────────────────────────────────────────────────────────────

function AIModuleSettings(): JSX.Element {
	const { t } = useTranslation(['aiModule']);
	const { user } = useAppContext();
	const isAdmin = user.role === USER_ROLES.ADMIN;

	const [isInitialLoading, setIsInitialLoading] = useState(true);
	const [isSaving, setIsSaving] = useState(false);
	const [isTesting, setIsTesting] = useState(false);
	const [updatedAt, setUpdatedAt] = useState<string>('');
	// Persistent indicator surfaced when the most recent Test detected an
	// auth failure (expired/invalid token). Cleared by a successful save or
	// a successful test re-run, or when the user starts editing a secret.
	const [authIssue, setAuthIssue] = useState<string | null>(null);
	const [testStatus, setTestStatus] = useState<'success' | 'failure' | null>(
		null,
	);
	// Mask refs track the *persisted* secret state, not the form state. They
	// are set on initial load from cfg.{apiKey,oauthToken} === API_KEY_UNCHANGED
	// and updated only after a successful save. Toggling the transport radio
	// must NOT reset these — the persisted secrets remain on disk until a save
	// commits a different shape, so a transient cli↔api visit should not wipe
	// them. buildPayload already gates each sentinel on the active transport,
	// so the masks don't leak across transports.
	const apiKeyMasked = useRef(false);
	const oauthTokenMasked = useRef(false);
	const [apiKeySaved, setApiKeySaved] = useState(false);
	const [oauthTokenSaved, setOauthTokenSaved] = useState(false);

	const { control, handleSubmit, watch, setValue, reset } = useForm<FormValues>(
		{
			defaultValues: {
				provider: 'local',
				llmProvider: 'claude',
				transport: 'api',
				model: '',
				apiKey: '',
				oauthToken: '',
				binaryPath: '',
				timeoutSeconds: '',
			},
		},
	);

	const provider = watch('provider');
	const transport = watch('transport');
	const isLLM = provider === 'llm';
	const isAPI = transport === 'api';

	useEffect(() => {
		void logEvent(ANALYTICS.PAGE_VIEWED, { role: user.role });
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	// Load config on mount
	useEffect(() => {
		let cancelled = false;
		(async (): Promise<void> => {
			try {
				const res = await getAIConfig();
				if (cancelled) return;
				const cfg = res.data;
				apiKeyMasked.current = cfg.apiKey === API_KEY_UNCHANGED;
				oauthTokenMasked.current = cfg.oauthToken === API_KEY_UNCHANGED;
				setApiKeySaved(apiKeyMasked.current);
				setOauthTokenSaved(oauthTokenMasked.current);
				reset({
					provider: cfg.provider ?? 'local',
					llmProvider: cfg.llmProvider ?? 'claude',
					transport: cfg.transport ?? 'api',
					model: cfg.model ?? '',
					// Always show empty in the field; mask flag tracks the real state.
					apiKey: '',
					oauthToken: '',
					binaryPath: cfg.binaryPath ?? '',
					timeoutSeconds: cfg.timeoutSeconds > 0 ? cfg.timeoutSeconds : '',
				});
				setUpdatedAt(cfg.updatedAt ?? '');
			} catch (err) {
				toast.error(getErrorMessage(err, t('toast_load_error')));
			} finally {
				if (!cancelled) setIsInitialLoading(false);
			}
		})();
		return (): void => {
			cancelled = true;
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	const buildPayload = useCallback(
		(values: FormValues): AIConfigPayload => {
			const maskedOrValue = (
				masked: boolean,
				value: string,
				active: boolean,
			): string => {
				if (!active) return '';
				if (masked && value === '') return API_KEY_UNCHANGED;
				return value;
			};
			return {
				provider: values.provider,
				llmProvider: values.llmProvider,
				transport: values.transport,
				model: values.model,
				apiKey: maskedOrValue(
					apiKeyMasked.current,
					values.apiKey,
					values.transport === 'api',
				),
				oauthToken: maskedOrValue(
					oauthTokenMasked.current,
					values.oauthToken,
					values.transport === 'cli',
				),
				binaryPath: values.binaryPath,
				timeoutSeconds:
					values.timeoutSeconds === '' ? 0 : Number(values.timeoutSeconds),
			};
		},
		[],
	);

	const onSave = useCallback(
		async (values: FormValues): Promise<void> => {
			setIsSaving(true);
			try {
				await updateAIConfig(buildPayload(values));
				toast.success(t('toast_save_success'));
				setAuthIssue(null);
				if (values.apiKey !== '') {
					apiKeyMasked.current = true;
					setApiKeySaved(true);
					setValue('apiKey', '');
				}
				if (values.oauthToken !== '') {
					oauthTokenMasked.current = true;
					setOauthTokenSaved(true);
					setValue('oauthToken', '');
				}
				setUpdatedAt(new Date().toISOString());
				void logEvent(ANALYTICS.SAVED, {
					provider: values.provider,
					llmProvider: values.provider === 'llm' ? values.llmProvider : undefined,
					transport: values.provider === 'llm' ? values.transport : undefined,
				});
			} catch (err) {
				toast.error(getErrorMessage(err, t('toast_save_error')));
			} finally {
				setIsSaving(false);
			}
		},
		[buildPayload, setValue],
	);

	const onTest = useCallback(async (): Promise<void> => {
		const values = watch() as FormValues;
		setIsTesting(true);
		setTestStatus(null);
		try {
			const res = await testAIConfig(buildPayload(values));
			const result = res.data;
			if (result.ok) {
				toast.success(truncate(result.headline ?? t('test_ok_fallback'), 80));
				setAuthIssue(null);
				setTestStatus('success');
			} else {
				toast.error(result.error ?? 'Test failed');
				setTestStatus('failure');
				if (result.errorKind === 'auth') {
					setAuthIssue(
						values.transport === 'cli'
							? t('auth_issue_cli')
							: t('auth_issue_api'),
					);
				} else {
					setAuthIssue(null);
				}
			}
			void logEvent(ANALYTICS.TESTED, {
				provider: values.provider,
				ok: result.ok,
				errorKind: result.errorKind,
			});
		} catch (err) {
			toast.error(getErrorMessage(err, t('toast_test_error')));
			setTestStatus('failure');
		} finally {
			setIsTesting(false);
		}
	}, [buildPayload, watch]);

	const handleApiKeyFocus = useCallback(() => {
		if (apiKeyMasked.current) {
			apiKeyMasked.current = false;
			setApiKeySaved(false);
			setValue('apiKey', '');
		}
		setAuthIssue(null);
	}, [setValue]);

	const handleOAuthTokenFocus = useCallback(() => {
		setAuthIssue(null);
		if (oauthTokenMasked.current) {
			oauthTokenMasked.current = false;
			setOauthTokenSaved(false);
			setValue('oauthToken', '');
		}
	}, [setValue]);

	if (isInitialLoading) {
		return <Spinner tip="Loading..." height="70vh" />;
	}

	// Inline test control rendered next to the active credential field (LLM) or
	// below the timeout field (local/mock). Tests the *current* form values, so
	// no save is required before testing.
	const renderTestControl = (): JSX.Element => (
		<div className="ai-module-settings__field ai-module-settings__test-inline">
			<Button onClick={onTest} loading={isTesting} disabled={isSaving || !isAdmin}>
				{t('btn_test')}
			</Button>
			{testStatus === 'success' && (
				<Tag color="success">{t('test_status_success')}</Tag>
			)}
			{testStatus === 'failure' && (
				<Tag color="error">{t('test_status_failure')}</Tag>
			)}
		</div>
	);

	return (
		<div className="ai-module-settings" data-testid="ai-module-settings">
			<header className="ai-module-settings__header">
				<h1 className="ai-module-settings__header-title">{t('header_title')}</h1>
				<p className="ai-module-settings__header-subtitle">
					{t('header_subtitle')}
				</p>
			</header>

			{authIssue && (
				<Alert
					type="warning"
					showIcon
					closable
					message={t('auth_issue_title')}
					description={authIssue}
					onClose={(): void => setAuthIssue(null)}
					style={{ marginBottom: 'var(--spacing-4)' }}
				/>
			)}

			<form onSubmit={handleSubmit(onSave)} autoComplete="off">
				<section className="ai-module-settings__card">
					<h3 className="ai-module-settings__card-title">
						<Badge color="secondary" variant="default">
							1
						</Badge>
						{t('provider_title')}
					</h3>
					<p className="ai-module-settings__card-description">
						{t('provider_description')}
					</p>

					<div className="ai-module-settings__field">
						<Controller
							name="provider"
							control={control}
							render={({ field }): JSX.Element => (
								<Radio.Group {...field} disabled={!isAdmin}>
									<Radio value="mock">Mock</Radio>
									<Radio value="local">Local</Radio>
									<Radio value="llm">LLM</Radio>
								</Radio.Group>
							)}
						/>
					</div>
				</section>

				{(isLLM || provider === 'local' || provider === 'mock') && (
					<section
						className="ai-module-settings__card"
						style={{ marginTop: 'var(--spacing-6)' }}
					>
						<h3 className="ai-module-settings__card-title">
							<Badge color="secondary" variant="default">
								2
							</Badge>
							{t('connection_title')}
						</h3>
						<p className="ai-module-settings__card-description">
							{isLLM
								? t('connection_description_llm')
								: t('connection_description_non_llm')}
						</p>

						{isLLM && (
							<div className="ai-module-settings__field">
								<label className="ai-module-settings__field-label">{t('field_llm_provider')}</label>
								<Controller
									name="llmProvider"
									control={control}
									render={({ field }): JSX.Element => (
										<Radio.Group {...field} disabled={!isAdmin}>
											<Radio value="claude">Claude</Radio>
											<Radio value="codex">Codex</Radio>
										</Radio.Group>
									)}
								/>
							</div>
						)}

						{isLLM && (
							<div className="ai-module-settings__field">
								<label className="ai-module-settings__field-label">{t('field_transport')}</label>
								<Controller
									name="transport"
									control={control}
									render={({ field }): JSX.Element => (
										<Radio.Group {...field} disabled={!isAdmin}>
											<Radio value="api">API</Radio>
											<Radio value="cli">CLI</Radio>
										</Radio.Group>
									)}
								/>
								<p className="ai-module-settings__field-hint">
									{t('transport_hint')}
								</p>
							</div>
						)}

						<div className="ai-module-settings__field">
							<label className="ai-module-settings__field-label">{t('field_model')}</label>
							<Controller
								name="model"
								control={control}
								render={({ field }): JSX.Element => (
									<Input
										{...field}
										placeholder={
											isLLM ? 'claude-sonnet-4-6 / gpt-5' : 'auto'
										}
										style={{ maxWidth: 360 }}
										disabled={!isAdmin}
									/>
								)}
							/>
						</div>

						{isLLM && isAPI && (
							<div className="ai-module-settings__field">
								<label className="ai-module-settings__field-label">
									{t('field_api_key')}
									{apiKeySaved && (
										<Tag color="success" style={{ marginLeft: 8 }}>
											{t('credential_saved')}
										</Tag>
									)}
								</label>
								<Controller
									name="apiKey"
									control={control}
									render={({ field }): JSX.Element => (
										<Input.Password
											{...field}
											placeholder={t('api_key_placeholder')}
											onFocus={handleApiKeyFocus}
											style={{ maxWidth: 360 }}
											disabled={!isAdmin}
										/>
									)}
								/>
								<p className="ai-module-settings__field-hint">
									{t('api_key_hint_before')}
									<code> &lt;unchanged&gt;</code>{t('api_key_hint_after')}
								</p>
								{renderTestControl()}
							</div>
						)}

						{isLLM && !isAPI && (
							<>
								<div className="ai-module-settings__field">
									<label className="ai-module-settings__field-label">
										{t('field_oauth_token')}
										{oauthTokenSaved && (
											<Tag color="success" style={{ marginLeft: 8 }}>
												{t('credential_saved')}
											</Tag>
										)}
									</label>
									<Controller
										name="oauthToken"
										control={control}
										render={({ field }): JSX.Element =>
											// Codex+CLI accepts either a raw OPENAI_API_KEY or the full
											// ~/.codex/auth.json content (ChatGPT subscription). JSON paste
											// needs multi-line, so use TextArea. Claude+CLI tokens are
											// single-line; keep Password masking there.
											watch('llmProvider') === 'codex' ? (
												<Input.TextArea
													{...field}
													placeholder={t('oauth_token_placeholder')}
													onFocus={handleOAuthTokenFocus}
													autoSize={{ minRows: 1, maxRows: 8 }}
													style={{ maxWidth: 560, fontFamily: 'monospace' }}
													disabled={!isAdmin}
												/>
											) : (
												<Input.Password
													{...field}
													placeholder={t('oauth_token_placeholder')}
													onFocus={handleOAuthTokenFocus}
													style={{ maxWidth: 360 }}
													disabled={!isAdmin}
												/>
											)
										}
									/>
									<p className="ai-module-settings__field-hint">
										{watch('llmProvider') === 'claude'
											? t('oauth_hint_claude')
											: t('oauth_hint_codex')}
									</p>
									{renderTestControl()}
								</div>

								<div className="ai-module-settings__field">
									<label className="ai-module-settings__field-label">{t('field_binary_path')}</label>
									<Controller
										name="binaryPath"
										control={control}
										render={({ field }): JSX.Element => (
											<Input
												{...field}
												placeholder={watch('llmProvider') === 'claude' ? 'claude' : 'codex'}
												style={{ maxWidth: 360 }}
												disabled={!isAdmin}
											/>
										)}
									/>
									<p className="ai-module-settings__field-hint">
										{t('binary_path_hint_before')} <code>claude</code> or <code>codex</code> {t('binary_path_hint_after')}
									</p>
								</div>
							</>
						)}

						<div className="ai-module-settings__field">
							<label className="ai-module-settings__field-label">{t('field_timeout')}</label>
							<Controller
								name="timeoutSeconds"
								control={control}
								render={({ field }): JSX.Element => (
									<Input
										{...field}
										type="number"
										min={0}
										placeholder={isLLM && !isAPI ? '120' : '15'}
										style={{ maxWidth: 120 }}
										disabled={!isAdmin}
										onChange={(e): void =>
											field.onChange(
												e.target.value === '' ? '' : Number(e.target.value),
											)
										}
									/>
								)}
							/>
							<p className="ai-module-settings__field-hint">
								{t('timeout_hint')}
							</p>
						</div>

						{/* local/mock have no credential field — surface Test here. */}
						{!isLLM && renderTestControl()}
					</section>
				)}

				<div
					className="ai-module-settings__actions"
					style={{ marginTop: 'var(--spacing-6)' }}
				>
					<Button
						type="primary"
						htmlType="submit"
						loading={isSaving}
						disabled={isTesting || !isAdmin}
					>
						{t('btn_save')}
					</Button>
					{updatedAt && (
						<span className="ai-module-settings__actions-meta">
							{t('last_updated')} {updatedAt}
						</span>
					)}
				</div>
			</form>
		</div>
	);
}

export default AIModuleSettings;
