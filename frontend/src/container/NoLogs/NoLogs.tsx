import { useTranslation } from 'react-i18next';
import { Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import history from 'lib/history';
import { ArrowUpRight } from 'lucide-react';
import { DataSource } from 'types/common/queryBuilder';
import DOCLINKS from 'utils/docLinks';
import { openInNewTab } from 'utils/navigation';

import eyesEmojiUrl from '@/assets/Images/eyesEmoji.svg';

import './NoLogs.styles.scss';

export default function NoLogs({
	dataSource,
}: {
	dataSource: DataSource;
}): JSX.Element {
	const { t } = useTranslation(['common']);
	const { isCloudUser: isCloudUserVal } = useGetTenantLicense();
	const dataSourceLabel = t(`data_source_${dataSource}`);

	const handleLinkClick = (
		e: React.MouseEvent<HTMLAnchorElement, MouseEvent>,
	): void => {
		e.preventDefault();
		e.stopPropagation();

		if (isCloudUserVal) {
			if (dataSource === DataSource.TRACES) {
				logEvent('Traces Explorer: Navigate to onboarding', {});
			} else if (dataSource === DataSource.LOGS) {
				logEvent('Logs Explorer: Navigate to onboarding', {});
			} else if (dataSource === DataSource.METRICS) {
				logEvent('Metrics Explorer: Navigate to onboarding', {});
			}
			let link;
			if (dataSource === DataSource.TRACES) {
				link = ROUTES.GET_STARTED_APPLICATION_MONITORING;
			} else if (dataSource === DataSource.METRICS) {
				link = ROUTES.GET_STARTED_WITH_CLOUD;
			} else {
				link = ROUTES.GET_STARTED_LOGS_MANAGEMENT;
			}
			history.push(link);
		} else if (dataSource === 'traces') {
			openInNewTab(DOCLINKS.TRACES_EXPLORER_EMPTY_STATE);
		} else if (dataSource === DataSource.METRICS) {
			openInNewTab(DOCLINKS.METRICS_EXPLORER_EMPTY_STATE);
		} else {
			openInNewTab(`${DOCLINKS.USER_GUIDE}${dataSource}/`);
		}
	};
	return (
		<div className="no-logs-container">
			<div className="no-logs-container-content">
				<img className="eyes-emoji" src={eyesEmojiUrl} alt="eyes emoji" />
				<Typography className="no-logs-text">
					{t('no_data_yet', { dataSource: dataSourceLabel })}
					<span className="sub-text">
						{' '}
						{t('data_show_up_here', { dataSource: dataSourceLabel })}
					</span>
				</Typography>

				<Typography.Link className="send-logs-link" onClick={handleLinkClick}>
					{t('send_data_to_signoz', { dataSource: dataSourceLabel })}{' '}
					<ArrowUpRight size={16} />
				</Typography.Link>
			</div>
		</div>
	);
}
