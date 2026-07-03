import { useTranslation } from 'react-i18next';

import { NocAlert, NocCounts } from '../types';

export interface SummaryBandProps {
	counts: NocCounts;
	incident: NocAlert | null;
	stableSince?: string;
}

interface CounterProps {
	label: string;
	value: number;
	tone: 'crit' | 'err' | 'warn' | 'ok' | 'zero';
}

function Counter({ label, value, tone }: CounterProps): JSX.Element {
	return (
		<div className={`noc-c2-counter noc-c2-${tone}`}>
			<span className="noc-c2-counter-val">{value}</span>
			<span className="noc-c2-counter-lbl">{label}</span>
		</div>
	);
}

export default function SummaryBand({
	counts,
	incident,
	stableSince,
}: SummaryBandProps): JSX.Element {
	const { t } = useTranslation('home');
	const anomaly =
		counts.critical > 0 || counts.warning > 0 || counts.alerts > 0;

	return (
		<div className="noc-c2-band">
			<div className="noc-c2-title">{t('noc_c2_title')}</div>
			<div className="noc-c2-counters">
				<Counter
					label={t('noc_c2_critical')}
					value={counts.critical}
					tone={counts.critical > 0 ? 'crit' : 'zero'}
				/>
				<Counter
					label={t('noc_c2_warning')}
					value={counts.warning}
					tone={counts.warning > 0 ? 'warn' : 'zero'}
				/>
				<Counter label={t('noc_c2_healthy')} value={counts.healthy} tone="ok" />
				<Counter
					label={t('noc_c2_alerts')}
					value={counts.alerts}
					tone={counts.alerts > 0 ? 'err' : 'zero'}
				/>
			</div>
			<div className="noc-c2-band-spacer" />
			{anomaly && incident ? (
				<div className="noc-c2-incident">
					<span className="noc-c2-incident-tag">{t('noc_c2_incident_tag')}</span>
					<span className="noc-c2-incident-title">{incident.title}</span>
					{incident.age ? (
						<span className="noc-c2-incident-age">{incident.age}</span>
					) : null}
				</div>
			) : (
				<div className="noc-c2-stable">
					<span className="noc-c2-stable-dot" />
					<span className="noc-c2-stable-title">{t('noc_c2_stable_title')}</span>
					{stableSince ? (
						<span className="noc-c2-stable-since">
							{t('noc_c2_stable_since', { age: stableSince })}
						</span>
					) : null}
				</div>
			)}
		</div>
	);
}
