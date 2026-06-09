import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from 'antd';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import history from 'lib/history';
import { ArrowRight, ArrowUpRight } from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { LicensePlatform } from 'types/api/licensesV3/getActive';
import { USER_ROLES } from 'types/roles';
import { openInNewTab } from 'utils/navigation';

import triangleRulerUrl from '@/assets/Icons/triangle-ruler.svg';

import { DOCS_LINKS } from '../constants';

// Shared empty state for the Home "Services" widgets (Traces & Metrics).
// `source` only distinguishes the analytics event; the UI is identical.
function ServicesEmptyState({ source }: { source: string }): JSX.Element {
	const { t } = useTranslation(['home', 'common']);
	const { user, activeLicense } = useAppContext();

	return (
		<div className="empty-state-container">
			<div className="empty-state-content-container">
				<div className="empty-state-content">
					<img
						src={triangleRulerUrl}
						alt="empty-alert-icon"
						className="empty-state-icon"
					/>

					<div className="empty-title">{t('home:services_empty_title')}</div>

					<div className="empty-description">
						{t('home:services_empty_description')}
					</div>
				</div>

				{user?.role !== USER_ROLES.VIEWER && (
					<div className="empty-actions-container">
						<Button
							type="default"
							className="periscope-btn secondary"
							onClick={(): void => {
								logEvent('Homepage: Get Started clicked', { source });

								if (
									activeLicense &&
									activeLicense.platform === LicensePlatform.CLOUD
								) {
									history.push(ROUTES.GET_STARTED_WITH_CLOUD);
								} else {
									openInNewTab(DOCS_LINKS.ADD_DATA_SOURCE);
								}
							}}
						>
							{t('common:get_started')} &nbsp; <ArrowRight size={16} />
						</Button>

						<Button
							type="link"
							className="learn-more-link"
							onClick={(): void => {
								logEvent('Homepage: Learn more clicked', { source });
								window.open(
									'https://signoz.io/docs/instrumentation/overview/',
									'_blank',
								);
							}}
						>
							{t('common:learn_more')} <ArrowUpRight size={12} />
						</Button>
					</div>
				)}
			</div>
		</div>
	);
}

export default memo(ServicesEmptyState);
