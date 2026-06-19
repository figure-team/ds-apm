import { Activity } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { NocHealth, NocServiceRow } from '../types';
import NocPanel from './NocPanel';

const HEALTH_CLASS: Record<NocHealth, string> = {
	healthy: 'ok',
	warning: 'warn',
	critical: 'crit',
};

function formatRps(rps: number): string {
	if (rps >= 1000) {
		return `${(rps / 1000).toFixed(1)}k`;
	}
	return rps >= 10 ? `${Math.round(rps)}` : rps.toFixed(1);
}

export interface ServiceHealthListProps {
	rows: NocServiceRow[];
	isLoading: boolean;
	isError: boolean;
}

export default function ServiceHealthList({
	rows,
	isLoading,
	isError,
}: ServiceHealthListProps): JSX.Element {
	const { t } = useTranslation('home');

	const renderBody = (): JSX.Element => {
		if (isLoading) {
			return <div className="noc-health-msg">{t('noc_health_loading')}</div>;
		}
		if (isError) {
			return <div className="noc-health-msg">{t('noc_health_error')}</div>;
		}
		if (rows.length === 0) {
			return <div className="noc-health-msg">{t('noc_health_empty')}</div>;
		}
		return (
			<div className="noc-health-list">
				<div className="noc-health-head">
					<span className="noc-health-name">{t('noc_health_col_service')}</span>
					<span className="noc-health-m">P99</span>
					<span className="noc-health-m">{t('noc_health_col_err')}</span>
					<span className="noc-health-m">RPS</span>
				</div>
				{rows.map((row) => (
					<div
						className={`noc-health-row ${HEALTH_CLASS[row.health]}`}
						key={row.name}
					>
						<span className="noc-health-name">
							<span className={`noc-dot noc-${HEALTH_CLASS[row.health]}`} />
							<span className="noc-health-nm">{row.name}</span>
						</span>
						<span className="noc-health-m">{Math.round(row.p99Ms)}ms</span>
						<span
							className={`noc-health-m${row.errPct >= 1 ? ' noc-health-bad' : ''}`}
						>
							{row.errPct.toFixed(2)}%
						</span>
						<span className="noc-health-m">{formatRps(row.rps)}</span>
					</div>
				))}
			</div>
		);
	};

	return (
		<NocPanel
			icon={<Activity size={13} />}
			title={t('noc_panel_health')}
			action={<span>{t('noc_action_services')}</span>}
			className="noc-health-panel"
		>
			{renderBody()}
		</NocPanel>
	);
}
