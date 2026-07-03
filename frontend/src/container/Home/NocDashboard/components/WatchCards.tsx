import ROUTES from 'constants/routes';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { useTranslation } from 'react-i18next';

import { NocHealth, NocServiceRow } from '../types';

const HEALTH_CLASS: Record<NocHealth, string> = {
	healthy: 'ok',
	warning: 'warn',
	critical: 'crit',
};

function fmtRps(rps: number): string {
	if (rps >= 1000) return `${(rps / 1000).toFixed(1)}k`;
	return rps >= 10 ? `${Math.round(rps)}` : rps.toFixed(1);
}

export interface WatchCardsProps {
	services: NocServiceRow[];
	mode: 'anomaly' | 'watch';
	overflowCount?: number;
}

export default function WatchCards({
	services,
	mode,
	overflowCount = 0,
}: WatchCardsProps): JSX.Element {
	const { t } = useTranslation('home');
	const { safeNavigate } = useSafeNavigate();

	return (
		<div className={`noc-c2-watch noc-c2-watch-${mode}`}>
			<div className="noc-c2-watch-head">
				<span className="noc-c2-watch-title">
					{mode === 'anomaly'
						? t('noc_c2_watch_anomaly')
						: t('noc_c2_watch_normal')}
				</span>
				{overflowCount > 0 ? (
					<button
						type="button"
						className="noc-c2-watch-overflow"
						onClick={(): void => safeNavigate(ROUTES.APPLICATION)}
					>
						{t('noc_c2_watch_overflow', { count: overflowCount })}
					</button>
				) : null}
			</div>
			<div className="noc-c2-watch-cards">
				{services.map((s) => (
					<button
						type="button"
						key={s.name}
						className={`noc-c2-card noc-${HEALTH_CLASS[s.health]}`}
						onClick={(): void =>
							safeNavigate(`${ROUTES.APPLICATION}/${s.name}`)
						}
					>
						<div className="noc-c2-card-top">
							<span className={`noc-c2-dot noc-${HEALTH_CLASS[s.health]}`} />
							<span className="noc-c2-card-name">{s.name}</span>
							<span className={`noc-badge noc-${HEALTH_CLASS[s.health]}`}>
								{t(`noc_c2_health_${s.health}`)}
							</span>
						</div>
						<div className="noc-c2-card-metrics">
							<span className="noc-c2-card-err">{s.errPct.toFixed(2)}%</span>
							<span className="noc-c2-card-sub">P99 {Math.round(s.p99Ms)}ms</span>
							<span className="noc-c2-card-sub">RPS {fmtRps(s.rps)}</span>
						</div>
					</button>
				))}
			</div>
		</div>
	);
}
