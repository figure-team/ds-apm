import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import { Button } from 'antd';

import {
	getRemediation,
	approveRemediation,
	rejectRemediation,
	RemediationExecution,
} from 'api/remediation';
import RemediationStatusBadge from 'components/Remediation/RemediationStatusBadge';
import RemediationResult from 'components/Remediation/RemediationResult';

import './styles.scss';

const POLL_INTERVAL_MS = 3000;
const TERMINAL = new Set([
	'verified', 'unresolved', 'failed', 'rejected', 'expired',
]);

type LoadError = 'not_found' | null;

function RemediationApprove(): JSX.Element {
	const { t } = useTranslation('alerts');
	const { id } = useParams<{ id: string }>();
	const [rem, setRem] = useState<RemediationExecution | null>(null);
	const [loadError, setLoadError] = useState<LoadError>(null);
	const [busy, setBusy] = useState(false);
	const [actionError, setActionError] = useState('');
	const [justApproved, setJustApproved] = useState(false);

	const load = useCallback(async (): Promise<RemediationExecution | null> => {
		try {
			const r = await getRemediation(id);
			setRem(r);
			return r;
		} catch {
			setLoadError('not_found');
			return null;
		}
	}, [id]);

	useEffect(() => {
		let active = true;
		let timer: ReturnType<typeof setTimeout> | undefined;
		const tick = async (): Promise<void> => {
			const r = await load();
			if (active && r && !TERMINAL.has(r.status) && r.status !== 'proposed') {
				timer = setTimeout(tick, POLL_INTERVAL_MS);
			}
		};
		tick();
		return (): void => {
			active = false;
			if (timer) clearTimeout(timer);
		};
	}, [load]);

	const httpStatus = (e: unknown): number | undefined =>
		(e as { response?: { status?: number } })?.response?.status;

	const onApprove = useCallback(async (): Promise<void> => {
		// eslint-disable-next-line no-alert
		if (!window.confirm(t('remediation_confirm'))) return;
		setBusy(true);
		setActionError('');
		try {
			await approveRemediation(id);
			setJustApproved(true);
		} catch (e) {
			const s = httpStatus(e);
			if (s === 403) setActionError(t('remediation_forbidden'));
			else if (s === 429) setActionError(t('remediation_too_many'));
			else await load(); // 409 등: 최신 상태로 갱신
		} finally {
			setBusy(false);
		}
	}, [id, t, load]);

	const onReject = useCallback(async (): Promise<void> => {
		setBusy(true);
		setActionError('');
		try {
			await rejectRemediation(id);
			await load();
		} catch (e) {
			if (httpStatus(e) === 403) setActionError(t('remediation_forbidden'));
			else await load();
		} finally {
			setBusy(false);
		}
	}, [id, t, load]);

	const renderBody = (): JSX.Element => {
		if (loadError === 'not_found') {
			return <p className="remediation-approve__msg">{t('remediation_not_found')}</p>;
		}
		if (!rem) {
			return <p className="remediation-approve__msg">{t('remediation_approving')}</p>;
		}
		if (justApproved) {
			return (
				<div className="remediation-approve__done">
					<h2>{t('remediation_approved_done_title')}</h2>
					<p>{t('remediation_approved_done_desc')}</p>
					<Button onClick={(): void => window.close()}>
						{t('remediation_close_window')}
					</Button>
				</div>
			);
		}
		if (rem.status === 'expired') {
			return (
				<div className="remediation-approve__msg">
					<h2>{t('remediation_expired_title')}</h2>
					<p>{t('remediation_expired_desc')}</p>
				</div>
			);
		}
		if (rem.status === 'rejected') {
			return <h2 className="remediation-approve__msg">{t('remediation_rejected_title')}</h2>;
		}
		if (rem.status === 'proposed') {
			return (
				<div className="remediation-approve__proposed">
					<pre className="remediation-card__script">{rem.scriptSnapshot}</pre>
					{actionError && <p className="remediation-approve__error">{actionError}</p>}
					<div className="remediation-approve__actions">
						<Button type="primary" disabled={busy} onClick={onApprove}>
							{t('remediation_approve')}
						</Button>
						<Button disabled={busy} onClick={onReject}>
							{t('remediation_reject')}
						</Button>
					</div>
				</div>
			);
		}
		// executing / succeeded / failed / verified / unresolved
		const title = rem.status === 'executing'
			? t('remediation_executing_title')
			: t('remediation_executed_title');
		return (
			<div className="remediation-approve__executed">
				<h2>{title}</h2>
				<RemediationResult rem={rem} />
			</div>
		);
	};

	return (
		<div className="remediation-approve">
			<div className="remediation-approve__card">
				<div className="remediation-approve__header">
					<h1>{t('remediation_approve_page_title')}</h1>
					{rem && <RemediationStatusBadge status={rem.status} />}
				</div>
				{renderBody()}
			</div>
		</div>
	);
}

export default RemediationApprove;
