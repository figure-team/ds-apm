import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import {
	getRemediation,
	approveRemediation,
	rejectRemediation,
	RemediationExecution,
} from 'api/remediation';
import RemediationStatusBadge from 'components/Remediation/RemediationStatusBadge';
import RemediationResult from 'components/Remediation/RemediationResult';

import './styles.scss';

const TERMINAL_STATUSES = new Set([
	'verified',
	'unresolved',
	'failed',
	'rejected',
	'expired',
]);

const POLL_INTERVAL_MS = 3000;

interface RemediationCardProps {
	remediationId: string;
}

function RemediationCard({ remediationId }: RemediationCardProps): JSX.Element | null {
	const { t } = useTranslation('alerts');
	const [rem, setRem] = useState<RemediationExecution | null>(null);
	const [busy, setBusy] = useState(false);

	const load = useCallback(async (): Promise<RemediationExecution> => {
		const r = await getRemediation(remediationId);
		setRem(r);
		return r;
	}, [remediationId]);

	useEffect(() => {
		let active = true;
		let timer: ReturnType<typeof setTimeout> | undefined;

		const tick = async (): Promise<void> => {
			try {
				const r = await load();
				if (active && !TERMINAL_STATUSES.has(r.status)) {
					timer = setTimeout(tick, POLL_INTERVAL_MS);
				}
			} catch {
				// silent: polling errors won't crash the component
			}
		};

		tick();

		return (): void => {
			active = false;
			if (timer !== undefined) {
				clearTimeout(timer);
			}
		};
	}, [load]);

	const onApprove = async (): Promise<void> => {
		// eslint-disable-next-line no-alert
		if (!window.confirm(t('remediation_confirm'))) return;
		setBusy(true);
		try {
			await approveRemediation(remediationId);
			await load();
		} finally {
			setBusy(false);
		}
	};

	const onReject = async (): Promise<void> => {
		setBusy(true);
		try {
			await rejectRemediation(remediationId);
			await load();
		} finally {
			setBusy(false);
		}
	};

	if (!rem) return null;

	const isProposed = rem.status === 'proposed';
	// Once the remediation has actually run it carries a result (exit code /
	// output / verify outcome). For those we lead with the result and tuck the
	// raw script behind a toggle — the executed code matters less than what it
	// did. Before that (proposed/approved/executing) the script is the point.
	const hasResult =
		typeof rem.exitCode === 'number' ||
		Boolean(rem.outputSnippet) ||
		Boolean(rem.verifyResult);

	return (
		<div className="remediation-card">
			<div className="remediation-card__header">
				<h4 className="remediation-card__title">
					{t(hasResult ? 'remediation_result_card_title' : 'remediation_card_title')}
				</h4>
				<RemediationStatusBadge status={rem.status} />
			</div>
			<div className="remediation-card__meta">
				{t('remediation_source_sop')}: {rem.sopId}
			</div>
			{hasResult ? (
				<RemediationResult rem={rem} />
			) : (
				<pre className="remediation-card__script">{rem.scriptSnapshot}</pre>
			)}
			{isProposed && (
				<div className="remediation-card__actions">
					<button
						type="button"
						className="remediation-card__btn remediation-card__btn--approve"
						disabled={busy}
						onClick={onApprove}
					>
						{t('remediation_approve')}
					</button>
					<button
						type="button"
						className="remediation-card__btn remediation-card__btn--reject"
						disabled={busy}
						onClick={onReject}
					>
						{t('remediation_reject')}
					</button>
				</div>
			)}
		</div>
	);
}

export default RemediationCard;
