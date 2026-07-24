import {
	type Dispatch,
	type SetStateAction,
	useCallback,
	useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import { Alert, Button, InputNumber, Select, Switch } from 'antd';
import updateConfig from 'api/codeRca/updateConfig';
import { CodeRcaConfig } from 'api/codeRca/types';

type Props = {
	config: CodeRcaConfig | null;
	setConfig: Dispatch<SetStateAction<CodeRcaConfig | null>>;
	isAdmin: boolean;
};

/** 기능 on/off와 실행 임계값(쿨다운·일일 상한·큐·동시성)을 다루는 카드. */
function RcaConfigCard({ config, setConfig, isAdmin }: Props): JSX.Element {
	const { t } = useTranslation(['codeRca']);
	const [isSaving, setIsSaving] = useState(false);

	const handleSaveConfig = useCallback(async (): Promise<void> => {
		if (!config) {
			return;
		}
		setIsSaving(true);
		try {
			await updateConfig(config);
			toast.success(t('saved'));
		} catch {
			toast.error(t('save_failed'));
		} finally {
			setIsSaving(false);
		}
	}, [config, t]);

	return (
		<section className="code-rca-settings__card">
			<h3 className="code-rca-settings__card-title">{t('field_enabled')}</h3>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_enabled')}
				</label>
				<Switch
					checked={config?.enabled ?? false}
					onChange={(val): void =>
						setConfig((prev) => (prev ? { ...prev, enabled: val } : prev))
					}
					disabled={!isAdmin}
					style={{ alignSelf: 'flex-start' }}
				/>
			</div>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_min_severity')}
				</label>
				<Select
					value={config?.minSeverity ?? 'error'}
					onChange={(val): void =>
						setConfig((prev) => (prev ? { ...prev, minSeverity: val } : prev))
					}
					disabled={!isAdmin}
					style={{ width: 180 }}
					options={[
						{ value: 'critical', label: 'critical' },
						{ value: 'error', label: 'error' },
						{ value: 'warning', label: 'warning' },
						{ value: 'info', label: 'info' },
					]}
				/>
			</div>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_cooldown')}
				</label>
				<InputNumber
					value={config?.cooldownWindowSecs ?? 0}
					min={0}
					onChange={(val): void =>
						setConfig((prev) =>
							prev ? { ...prev, cooldownWindowSecs: val ?? 0 } : prev,
						)
					}
					disabled={!isAdmin}
				/>
			</div>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_max_runs_per_day')}
				</label>
				<InputNumber
					value={config?.maxRunsPerDay ?? 0}
					min={0}
					onChange={(val): void =>
						setConfig((prev) => (prev ? { ...prev, maxRunsPerDay: val ?? 0 } : prev))
					}
					disabled={!isAdmin}
				/>
			</div>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_max_queue_depth')}
				</label>
				<InputNumber
					value={config?.maxQueueDepth ?? 0}
					min={0}
					onChange={(val): void =>
						setConfig((prev) => (prev ? { ...prev, maxQueueDepth: val ?? 0 } : prev))
					}
					disabled={!isAdmin}
				/>
			</div>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_max_concurrent')}
				</label>
				<InputNumber
					value={config?.maxConcurrentRuns ?? 0}
					min={0}
					onChange={(val): void =>
						setConfig((prev) =>
							prev ? { ...prev, maxConcurrentRuns: val ?? 0 } : prev,
						)
					}
					disabled={!isAdmin}
				/>
			</div>

			<div className="code-rca-settings__field">
				<label className="code-rca-settings__field-label">
					{t('field_allow_unbound')}
				</label>
				<Switch
					checked={config?.allowUnboundWithoutAnomaly ?? false}
					onChange={(val): void =>
						setConfig((prev) =>
							prev ? { ...prev, allowUnboundWithoutAnomaly: val } : prev,
						)
					}
					disabled={!isAdmin}
					style={{ alignSelf: 'flex-start' }}
				/>
				{config?.allowUnboundWithoutAnomaly && (
					<Alert
						type="warning"
						showIcon
						message={t('allow_unbound_warning')}
						style={{ marginTop: 8 }}
					/>
				)}
			</div>

			<div className="code-rca-settings__actions">
				<Button
					type="primary"
					onClick={handleSaveConfig}
					loading={isSaving}
					disabled={!isAdmin}
				>
					{t('save')}
				</Button>
			</div>
		</section>
	);
}

export default RcaConfigCard;
