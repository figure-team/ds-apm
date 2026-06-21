import ROUTES from 'constants/routes';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { BarChart3, Siren, Zap } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { NocAccent, NocAlert, NocGoldenSignal, NocKpi, NocSeverity } from '../types';
import NocPanel from './NocPanel';
import Sparkline from './Sparkline';

const ACCENT_VAR: Record<NocAccent, string> = {
	brand: 'var(--noc-crit)',
	error: 'var(--noc-err)',
	ok: 'var(--noc-ok)',
	neutral: 'var(--noc-c-sec)',
};

const SEVERITY_CLASS: Record<NocSeverity, string> = {
	critical: 'noc-crit',
	error: 'noc-err',
	warning: 'noc-warn',
	info: 'noc-info',
};

export interface LeftRailProps {
	kpis: NocKpi[];
	goldenSignals: NocGoldenSignal[];
	alerts: NocAlert[];
	alertsLoading: boolean;
	alertsError: boolean;
}

export default function LeftRail({
	kpis,
	goldenSignals,
	alerts,
	alertsLoading,
	alertsError,
}: LeftRailProps): JSX.Element {
	const { t } = useTranslation('home');
	const { safeNavigate } = useSafeNavigate();

	const renderAlerts = (): JSX.Element => {
		if (alertsLoading) {
			return <div className="noc-empty">{t('noc_alerts_loading')}</div>;
		}
		if (alertsError) {
			return <div className="noc-empty">{t('noc_alerts_error')}</div>;
		}
		if (alerts.length === 0) {
			return <div className="noc-empty">{t('noc_alerts_empty')}</div>;
		}
		return (
			<>
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
			</>
		);
	};

	return (
		<div className="noc-rail">
			<NocPanel icon={<BarChart3 size={13} />} title={t('noc_panel_kpis')}>
				<div className="noc-kpi-stack">
					{kpis.map((kpi) => (
						<div className="noc-kpi" key={kpi.key}>
							<div className="noc-kpi-lbl">{t(`noc_kpi_${kpi.key}`, kpi.label)}</div>
							<div
								className="noc-kpi-val"
								style={
									kpi.accent && kpi.accent !== 'neutral'
										? { color: ACCENT_VAR[kpi.accent] }
										: undefined
								}
							>
								{kpi.value}
								{kpi.unit ? <small>{kpi.unit}</small> : null}
							</div>
							{kpi.delta ? (
								<div className={`noc-kpi-d noc-d-${kpi.deltaDir ?? 'flat'}`}>
									{kpi.delta}
								</div>
							) : null}
							{kpi.spark ? (
								<Sparkline
									points={kpi.spark}
									color={kpi.accent ? ACCENT_VAR[kpi.accent] : 'var(--noc-c-sec)'}
								/>
							) : null}
						</div>
					))}
				</div>
			</NocPanel>

			<NocPanel icon={<Zap size={13} />} title={t('noc_panel_golden')}>
				<div className="noc-golden">
					{goldenSignals.map((sig) => (
						<div className="noc-golden-item" key={sig.key}>
							<div className="noc-golden-lbl">{t(`noc_golden_${sig.key}`, sig.label)}</div>
							<div
								className="noc-golden-val"
								style={
									sig.accent === 'error' ? { color: 'var(--noc-err)' } : undefined
								}
							>
								{sig.value}
							</div>
						</div>
					))}
				</div>
			</NocPanel>

			<NocPanel
				icon={<Siren size={13} />}
				title={t('noc_panel_alerts')}
				action={<span>{t('noc_action_view_all')}</span>}
				onActionClick={(): void => safeNavigate(ROUTES.LIST_ALL_ALERT)}
			>
				{renderAlerts()}
			</NocPanel>
		</div>
	);
}
