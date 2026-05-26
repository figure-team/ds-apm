import { useCallback, useEffect, useRef, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Badge, toast } from '@signozhq/ui';
import { Alert, Button, Input, Radio } from 'antd';
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
	// Mask refs track the *persisted* secret state, not the form state. They
	// are set on initial load from cfg.{apiKey,oauthToken} === API_KEY_UNCHANGED
	// and updated only after a successful save. Toggling the transport radio
	// must NOT reset these — the persisted secrets remain on disk until a save
	// commits a different shape, so a transient cli↔api visit should not wipe
	// them. buildPayload already gates each sentinel on the active transport,
	// so the masks don't leak across transports.
	const apiKeyMasked = useRef(false);
	const oauthTokenMasked = useRef(false);

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
				toast.error(getErrorMessage(err, 'Failed to load AI config'));
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
				toast.success('AI module configuration saved');
				setAuthIssue(null);
				if (values.apiKey !== '') {
					apiKeyMasked.current = true;
					setValue('apiKey', '');
				}
				if (values.oauthToken !== '') {
					oauthTokenMasked.current = true;
					setValue('oauthToken', '');
				}
				setUpdatedAt(new Date().toISOString());
				void logEvent(ANALYTICS.SAVED, {
					provider: values.provider,
					llmProvider: values.provider === 'llm' ? values.llmProvider : undefined,
					transport: values.provider === 'llm' ? values.transport : undefined,
				});
			} catch (err) {
				toast.error(getErrorMessage(err, 'Failed to save AI config'));
			} finally {
				setIsSaving(false);
			}
		},
		[buildPayload, setValue],
	);

	const onTest = useCallback(async (): Promise<void> => {
		const values = watch() as FormValues;
		setIsTesting(true);
		try {
			const res = await testAIConfig(buildPayload(values));
			const result = res.data;
			if (result.ok) {
				toast.success(truncate(result.headline ?? 'Connection OK', 80));
				setAuthIssue(null);
			} else {
				toast.error(result.error ?? 'Test failed');
				if (result.errorKind === 'auth') {
					setAuthIssue(
						values.transport === 'cli'
							? 'The configured OAuth token was rejected. Re-paste a fresh token (claude setup-token / codex login).'
							: 'The configured API key was rejected. Re-paste a fresh key from the provider console.',
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
			toast.error(getErrorMessage(err, 'Test request failed'));
		} finally {
			setIsTesting(false);
		}
	}, [buildPayload, watch]);

	const handleApiKeyFocus = useCallback(() => {
		if (apiKeyMasked.current) {
			apiKeyMasked.current = false;
			setValue('apiKey', '');
		}
		setAuthIssue(null);
	}, [setValue]);

	const handleOAuthTokenFocus = useCallback(() => {
		setAuthIssue(null);
		if (oauthTokenMasked.current) {
			oauthTokenMasked.current = false;
			setValue('oauthToken', '');
		}
	}, [setValue]);

	if (isInitialLoading) {
		return <Spinner tip="Loading..." height="70vh" />;
	}

	return (
		<div className="ai-module-settings" data-testid="ai-module-settings">
			<header className="ai-module-settings__header">
				<h1 className="ai-module-settings__header-title">AI Module</h1>
				<p className="ai-module-settings__header-subtitle">
					Configure which backend generates incident-response strategies for SOP-bound
					alerts. Mock and Local modes need no external setup; LLM mode calls an
					external API or a local CLI.
				</p>
			</header>

			{authIssue && (
				<Alert
					type="warning"
					showIcon
					closable
					message="Authentication issue detected"
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
						Provider
					</h3>
					<p className="ai-module-settings__card-description">
						Pick which generator runs when alertmanager dispatches an alert. Mock returns
						a scripted scenario response. Local runs the deterministic in-process
						generator. LLM hands off to an external model.
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
							Connection
						</h3>
						<p className="ai-module-settings__card-description">
							{isLLM
								? 'Choose the LLM vendor and transport, then provide the model identifier and credential. Use the Test button to probe the configuration without saving.'
								: 'Optional overrides. Leave the model blank to use the package default.'}
						</p>

						{isLLM && (
							<div className="ai-module-settings__field">
								<label className="ai-module-settings__field-label">LLM Provider</label>
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
								<label className="ai-module-settings__field-label">Transport</label>
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
									API uses HTTPS with the configured API key. CLI shells out to the
									local binary; the container must have it installed and authenticated.
								</p>
							</div>
						)}

						<div className="ai-module-settings__field">
							<label className="ai-module-settings__field-label">Model</label>
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
								<label className="ai-module-settings__field-label">API Key</label>
								<Controller
									name="apiKey"
									control={control}
									render={({ field }): JSX.Element => (
										<Input.Password
											{...field}
											placeholder="Leave blank to keep the existing key"
											onFocus={handleApiKeyFocus}
											style={{ maxWidth: 360 }}
											disabled={!isAdmin}
										/>
									)}
								/>
								<p className="ai-module-settings__field-hint">
									Stored encrypted at rest (AES-GCM). Returned to the UI as
									<code> &lt;unchanged&gt;</code>; type a new value to overwrite.
								</p>
							</div>
						)}

						{isLLM && !isAPI && (
							<>
								<div className="ai-module-settings__field">
									<label className="ai-module-settings__field-label">OAuth Token</label>
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
													placeholder="Leave blank to keep the existing token"
													onFocus={handleOAuthTokenFocus}
													autoSize={{ minRows: 1, maxRows: 8 }}
													style={{ maxWidth: 560, fontFamily: 'monospace' }}
													disabled={!isAdmin}
												/>
											) : (
												<Input.Password
													{...field}
													placeholder="Leave blank to keep the existing token"
													onFocus={handleOAuthTokenFocus}
													style={{ maxWidth: 360 }}
													disabled={!isAdmin}
												/>
											)
										}
									/>
									<p className="ai-module-settings__field-hint">
										{watch('llmProvider') === 'claude'
											? 'Run `claude setup-token` on a machine with a browser, sign in, then paste the issued token here. Stored encrypted at rest (AES-GCM).'
											: 'Two options: paste an OPENAI_API_KEY (single line) for direct API auth, OR paste the full `~/.codex/auth.json` JSON (multi-line) for ChatGPT subscription auth. Stored encrypted at rest (AES-GCM).'}
									</p>
								</div>

								<div className="ai-module-settings__field">
									<label className="ai-module-settings__field-label">Binary path</label>
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
										Defaults to <code>claude</code> or <code>codex</code> on the container's PATH.
									</p>
								</div>
							</>
						)}

						<div className="ai-module-settings__field">
							<label className="ai-module-settings__field-label">Timeout (seconds)</label>
							<Controller
								name="timeoutSeconds"
								control={control}
								render={({ field }): JSX.Element => (
									<Input
										{...field}
										type="number"
										min={0}
										placeholder="15"
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
								Per-call deadline. 0 picks the package default (15 s).
							</p>
						</div>
					</section>
				)}

				<div
					className="ai-module-settings__actions"
					style={{ marginTop: 'var(--spacing-6)' }}
				>
					<Button onClick={onTest} loading={isTesting} disabled={isSaving || !isAdmin}>
						Test
					</Button>
					<Button
						type="primary"
						htmlType="submit"
						loading={isSaving}
						disabled={isTesting || !isAdmin}
					>
						Save changes
					</Button>
					{updatedAt && (
						<span className="ai-module-settings__actions-meta">
							Last updated · {updatedAt}
						</span>
					)}
				</div>
			</form>
		</div>
	);
}

export default AIModuleSettings;
