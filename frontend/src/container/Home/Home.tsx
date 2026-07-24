import { Trans, useTranslation } from 'react-i18next';
import { House } from '@signozhq/icons';
import { PersistedAnnouncementBanner } from '@signozhq/ui';
import Header from 'components/Header/Header';
import { LOCALSTORAGE } from 'constants/localStorage';
import ROUTES from 'constants/routes';
import history from 'lib/history';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import OnboardingHome from './components/OnboardingHome';
import useHomeIngestionStatus from './hooks/useHomeIngestionStatus';
import NocDashboard from './NocDashboard/NocDashboard';

import './Home.styles.scss';

export default function Home(): JSX.Element {
	const { t } = useTranslation('home');
	const { user } = useAppContext();
	const {
		isAnyIngestionActive,
		showNocDashboard,
		isLogsLoading,
		isTracesLoading,
	} = useHomeIngestionStatus();

	return (
		<div className="home-container">
			{user?.role === USER_ROLES.ADMIN && (
				<PersistedAnnouncementBanner
					type="info"
					storageKey={LOCALSTORAGE.DISMISSED_API_KEYS_DEPRECATION_BANNER}
					action={{
						label: t('go_to_service_accounts'),
						onClick: (): void => history.push(ROUTES.SERVICE_ACCOUNTS_SETTINGS),
					}}
				>
					<Trans
						t={t}
						i18nKey="api_keys_deprecated"
						components={[<strong key="0" />, <strong key="1" />]}
					/>
				</PersistedAnnouncementBanner>
			)}

			<div className="sticky-header">
				<Header
					leftComponent={
						<div className="home-header-left">
							<House size={14} /> {t('page_title')}
						</div>
					}
					rightComponent={null}
				/>
			</div>

			<div
				className={`home-content${showNocDashboard ? ' home-content--noc' : ''}`}
			>
				{showNocDashboard ? (
					<NocDashboard />
				) : (
					<OnboardingHome
						isAnyIngestionActive={isAnyIngestionActive}
						isLogsLoading={isLogsLoading}
						isTracesLoading={isTracesLoading}
					/>
				)}
			</div>
		</div>
	);
}
