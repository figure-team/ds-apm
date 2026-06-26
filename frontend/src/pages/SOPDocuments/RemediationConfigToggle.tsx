import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import { Switch } from 'antd';
import {
	getRemediationConfig,
	RemediationConfig,
	updateRemediationConfig,
} from 'api/remediation';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

// RemediationConfigToggle is the org-wide auto-remediation master switch shown
// under the SOP binding preview. Admin-only: non-admins don't see the control
// at all (the backend GET/PUT /remediation/config routes also enforce
// AdminAccess, so hiding here is UX, not the security boundary).
function RemediationConfigToggle(): JSX.Element | null {
	const { t } = useTranslation(['sop_documents']);
	const { user } = useAppContext();
	const isAdmin = user.role === USER_ROLES.ADMIN;

	const [config, setConfig] = useState<RemediationConfig | null>(null);
	const [loading, setLoading] = useState(false);
	const [saving, setSaving] = useState(false);

	useEffect(() => {
		if (!isAdmin) return undefined;
		let active = true;
		setLoading(true);
		getRemediationConfig()
			.then((cfg) => {
				if (active) setConfig(cfg);
			})
			.catch(() => {
				if (active) toast.error(t('remediation_toggle_load_error'));
			})
			.finally(() => {
				if (active) setLoading(false);
			});
		return (): void => {
			active = false;
		};
	}, [isAdmin, t]);

	const handleToggle = useCallback(
		async (checked: boolean): Promise<void> => {
			if (!config) return;
			const message = checked
				? t('remediation_toggle_confirm_on')
				: t('remediation_toggle_confirm_off');
			// eslint-disable-next-line no-alert
			if (!window.confirm(message)) return;
			setSaving(true);
			try {
				const next = await updateRemediationConfig({
					...config,
					executionEnabled: checked,
				});
				setConfig(next);
			} catch {
				toast.error(t('remediation_toggle_save_error'));
			} finally {
				setSaving(false);
			}
		},
		[config, t],
	);

	// Admin-only visibility.
	if (!isAdmin) return null;

	return (
		<div
			className="sop-documents-page__remediation-toggle"
			data-testid="remediation-config-toggle"
		>
			<div className="sop-documents-page__remediation-toggle-head">
				<Switch
					checked={Boolean(config?.executionEnabled)}
					data-testid="remediation-execution-switch"
					disabled={loading || saving || !config}
					loading={loading || saving}
					onChange={handleToggle}
				/>
				<div className="sop-documents-page__remediation-toggle-text">
					<h3>{t('remediation_toggle_title')}</h3>
					<p>{t('remediation_toggle_description')}</p>
				</div>
				{config && (
					<span
						className={`sop-documents-page__remediation-toggle-status sop-documents-page__remediation-toggle-status--${
							config.executionEnabled ? 'on' : 'off'
						}`}
					>
						{config.executionEnabled
							? t('remediation_toggle_status_enabled')
							: t('remediation_toggle_status_disabled')}
					</span>
				)}
			</div>
		</div>
	);
}

export default RemediationConfigToggle;
