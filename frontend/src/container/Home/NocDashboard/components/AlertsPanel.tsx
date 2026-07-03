import ROUTES from 'constants/routes';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { CheckCircle2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { NocAlert, NocSeverity } from '../types';

const SEVERITY_CLASS: Record<NocSeverity, string> = {
	critical: 'noc-crit',
	error: 'noc-err',
	warning: 'noc-warn',
	info: 'noc-info',
};

export interface AlertsPanelProps {
	alerts: NocAlert[];
	isLoading: boolean;
	isError: boolean;
	lastResolved?: { age: string; service: string };
}

export default function AlertsPanel({
	alerts,
	isLoading,
	isError,
	lastResolved,
}: AlertsPanelProps): JSX.Element {
	const { t } = useTranslation('home');
	const { safeNavigate } = useSafeNavigate();

	if (isLoading) return <div className="noc-empty">{t('noc_alerts_loading')}</div>;
	if (isError) return <div className="noc-empty">{t('noc_alerts_error')}</div>;
	if (alerts.length === 0) {
		return (
			<div className="noc-c2-alerts-empty">
				<CheckCircle2 className="noc-c2-alerts-empty-icon" size={28} />
				<div className="noc-c2-alerts-empty-title">{t('noc_c2_alerts_empty')}</div>
				{lastResolved ? (
					<div className="noc-c2-alerts-empty-sub">
						{t('noc_c2_alerts_resolved', {
							age: lastResolved.age,
							service: lastResolved.service,
						})}
					</div>
				) : null}
			</div>
		);
	}
	return (
		<div className="noc-alert-list">
			{alerts.map((alert) => (
				<button
					type="button"
					className="noc-alert noc-alert-btn"
					key={alert.id}
					onClick={(): void =>
						safeNavigate(`${ROUTES.ALERT_OVERVIEW}?ruleId=${alert.id}`)
					}
				>
					<div className={`noc-alert-bar ${SEVERITY_CLASS[alert.severity]}`} />
					<div className="noc-alert-body">
						<div className="noc-alert-top">
							<span className={`noc-badge ${SEVERITY_CLASS[alert.severity]}`}>
								{t(`noc_sev_${alert.severity}`)}
							</span>
							<span className="noc-alert-msg">{alert.title}</span>
						</div>
						{alert.meta ? <div className="noc-alert-meta">{alert.meta}</div> : null}
					</div>
					{alert.age ? <div className="noc-alert-time">{alert.age}</div> : null}
				</button>
			))}
		</div>
	);
}
