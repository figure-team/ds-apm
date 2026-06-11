import { useCallback } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Button } from '@signozhq/ui';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import history from 'lib/history';
import { AlertTriangle, LifeBuoy } from 'lucide-react';

import emptyStateUrl from '@/assets/Icons/emptyState.svg';
import eyesEmojiUrl from '@/assets/Images/eyesEmoji.svg';

import { K8sBaseListEmptyStateContext } from './K8sBaseList';

import styles from './K8sEmptyState.module.scss';

export interface K8sListResponseMetadata {
	sentAnyHostMetricsData?: boolean;
	isSendingK8SAgentMetrics?: boolean;
	endTimeBeforeRetention?: boolean;
}

type K8sEmptyStateProps = Partial<K8sBaseListEmptyStateContext>;

const handleContactSupport = (isCloudUser: boolean): void => {
	if (isCloudUser) {
		history.push('/support');
	} else {
		window.open('https://signoz.io/slack', '_blank');
	}
};

export function K8sEmptyState({
	isError,
	error,
	isLoading,
	rawData,
}: K8sEmptyStateProps): JSX.Element | null {
	const { t } = useTranslation('infraMonitoring');
	const { isCloudUser } = useGetTenantLicense();

	const handleSupport = useCallback(() => {
		handleContactSupport(isCloudUser);
	}, [isCloudUser]);

	if (isLoading) {
		return null;
	}

	if (isError || error) {
		return (
			<div className={styles.container}>
				<div className={styles.content}>
					<AlertTriangle size={32} className={styles.errorIcon} />
					<span className={styles.message}>
						{error || 'An error occurred while fetching data.'}
					</span>
					<p>{t('error_team_resolving')}</p>
					<div className={styles.actions}>
						<Button
							onClick={handleSupport}
							variant="solid"
							color="secondary"
							prefix={<LifeBuoy size={14} />}
						>
							{t('contact_support')}
						</Button>
					</div>
				</div>
			</div>
		);
	}

	const metadata = rawData as K8sListResponseMetadata | undefined;

	if (metadata?.sentAnyHostMetricsData === false) {
		return (
			<div className={styles.container}>
				<div className={styles.content}>
					<img className={styles.eyesEmoji} src={eyesEmojiUrl} alt={t('eyes_emoji_alt')} />
					<div className={styles.noDataMessage}>
						<h5 className={styles.title}>{t('no_host_metrics_yet')}</h5>
						<span className={styles.message}>
							<Trans
								i18nKey="send_host_metrics_help"
								t={t}
								components={[
									// eslint-disable-next-line jsx-a11y/anchor-has-content
									<a
										key="0"
										href="https://signoz.io/docs/userguide/hostmetrics/"
										target="_blank"
										rel="noreferrer"
									>
										{' '}
									</a>,
								]}
							/>
						</span>
					</div>
				</div>
			</div>
		);
	}

	if (metadata?.isSendingK8SAgentMetrics) {
		return (
			<div className={styles.container}>
				<div className={styles.content}>
					<img className={styles.eyesEmoji} src={eyesEmojiUrl} alt={t('eyes_emoji_alt')} />
					<span className={styles.message}>
						{t('upgrade_k8s_infra_chart')}
					</span>
				</div>
			</div>
		);
	}

	if (metadata?.endTimeBeforeRetention) {
		return (
			<div className={styles.container}>
				<div className={styles.content}>
					<img className={styles.eyesEmoji} src={eyesEmojiUrl} alt={t('eyes_emoji_alt')} />
					<div className={styles.noDataMessage}>
						<h5 className={styles.title}>
							{t('queried_range_before_earliest')}
						</h5>
						<span className={styles.message}>
							{t('end_time_before_earliest')}
						</span>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className={styles.container}>
			<div className={styles.content}>
				<img
					src={emptyStateUrl}
					alt="empty-state"
					className={styles.emptyStateSvg}
				/>
				<span className={styles.message}>
					{t('query_no_results')}
				</span>
			</div>
		</div>
	);
}
