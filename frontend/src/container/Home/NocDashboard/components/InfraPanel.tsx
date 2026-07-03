import { useTranslation } from 'react-i18next';

import { NocHealth, NocInfraHost } from '../types';

const HEALTH_CLASS: Record<NocHealth, string> = {
	healthy: 'ok',
	warning: 'warn',
	critical: 'crit',
};

export interface InfraPanelProps {
	hosts: NocInfraHost[];
	isLoading: boolean;
	isError: boolean;
}

export default function InfraPanel({
	hosts,
	isLoading,
	isError,
}: InfraPanelProps): JSX.Element {
	const { t } = useTranslation('home');

	if (isLoading) return <div className="noc-empty">{t('noc_c2_infra_loading')}</div>;
	if (isError) return <div className="noc-empty">{t('noc_c2_infra_error')}</div>;
	if (hosts.length === 0) {
		return <div className="noc-empty">{t('noc_c2_infra_empty')}</div>;
	}

	return (
		<div className="noc-c2-infra-grid">
			{hosts.map((h) => (
				<div key={h.name} className={`noc-c2-infra-tile noc-${HEALTH_CLASS[h.health]}`}>
					<div className="noc-c2-infra-name">{h.name}</div>
					<div className="noc-c2-infra-metrics">
						<span className="noc-c2-infra-cpu">{h.cpu}%</span>
						<span className="noc-c2-infra-sub">MEM {h.mem}%</span>
					</div>
					<div className="noc-c2-infra-bar">
						<span
							className={`noc-c2-infra-fill noc-${HEALTH_CLASS[h.health]}`}
							style={{ width: `${Math.min(100, h.cpu)}%` }}
						/>
					</div>
				</div>
			))}
		</div>
	);
}
