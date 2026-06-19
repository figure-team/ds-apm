import { ScrollText, Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { NocLogLine, NocRca } from '../types';
import NocPanel from './NocPanel';

const LEVEL_CLASS: Record<NocLogLine['level'], string> = {
	ERROR: 'noc-lv-e',
	WARN: 'noc-lv-w',
	INFO: 'noc-lv-i',
	DEBUG: 'noc-lv-d',
};

export interface InsightsColumnProps {
	rca: NocRca | null;
	rcaLoading: boolean;
	rcaError: boolean;
	logs: NocLogLine[];
	logsLoading: boolean;
	logsError: boolean;
}

export default function InsightsColumn({
	rca,
	rcaLoading,
	rcaError,
	logs,
	logsLoading,
	logsError,
}: InsightsColumnProps): JSX.Element {
	const { t } = useTranslation('home');

	const renderRca = (): JSX.Element => {
		if (rcaLoading) {
			return <div className="noc-empty">{t('noc_rca_loading')}</div>;
		}
		if (rcaError) {
			return <div className="noc-empty">{t('noc_rca_error')}</div>;
		}
		if (!rca) {
			return <div className="noc-empty">{t('noc_rca_empty')}</div>;
		}
		return (
			<>
				<div className="noc-rca">
					<div className="noc-rca-head">
						<span className="noc-rca-ai">AI</span>
						{rca.title}
					</div>
					<div className="noc-rca-desc">{rca.summary}</div>
					{rca.chips.length > 0 ? (
						<div className="noc-rca-chips">
							{rca.chips.map((chip) => (
								<span className="noc-chip" key={chip}>
									{chip}
								</span>
							))}
						</div>
					) : null}
				</div>
				{rca.actions.length > 0 ? (
					<div className="noc-rca-actions">
						{rca.actions.map((action, i) => (
							<span
								className={`noc-chip${i === 0 ? ' noc-chip-brand' : ''}`}
								key={action}
							>
								{action}
							</span>
						))}
					</div>
				) : null}
			</>
		);
	};

	const renderLogs = (): JSX.Element => {
		if (logsLoading) {
			return <div className="noc-empty">{t('noc_logs_loading')}</div>;
		}
		if (logsError) {
			return <div className="noc-empty">{t('noc_logs_error')}</div>;
		}
		if (logs.length === 0) {
			return <div className="noc-empty">{t('noc_logs_empty')}</div>;
		}
		return (
			<div className="noc-logs">
				{logs.map((line, i) => (
					<div className="noc-logline" key={`${line.ts}-${line.service}-${i}`}>
						<span className={`noc-lv ${LEVEL_CLASS[line.level]}`}>{line.level}</span>
						{line.service ? (
							<span className="noc-log-svc">{line.service}</span>
						) : null}
						<span className="noc-log-tx">{line.message}</span>
					</div>
				))}
			</div>
		);
	};

	return (
		<div className="noc-col">
			<NocPanel icon={<Sparkles size={13} />} title={t('noc_panel_rca')}>
				{renderRca()}
			</NocPanel>

			<NocPanel
				icon={<ScrollText size={13} />}
				title={t('noc_panel_logs')}
				action={<span>{t('noc_action_explorer')}</span>}
			>
				{renderLogs()}
			</NocPanel>
		</div>
	);
}
