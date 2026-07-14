/* eslint-disable sonarjs/no-duplicate-string */
import React, { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Compass, Dot, House, Plus, Wrench } from '@signozhq/icons';
import { Button, PersistedAnnouncementBanner } from '@signozhq/ui';
import logEvent from 'api/common/logEvent';
import { useGetMetricsOnboardingStatus } from 'api/generated/services/metrics';
import Header from 'components/Header/Header';
import { ENTITY_VERSION_V5 } from 'constants/app';
import { LOCALSTORAGE } from 'constants/localStorage';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import ROUTES from 'constants/routes';
import { DEFAULT_TIME_RANGE } from 'container/TopNav/DateTimeSelectionV2/constants';
import { useGetQueryRange } from 'hooks/queryBuilder/useGetQueryRange';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import history from 'lib/history';
import Card from 'periscope/components/Card/Card';
import { useAppContext } from 'providers/App/App';
import { DataSource } from 'types/common/queryBuilder';
import { USER_ROLES } from 'types/roles';
import { isIngestionActive } from 'utils/app';
import { isModifierKeyPressed } from 'utils/app';

import crackerUrl from '@/assets/Icons/cracker.svg';
import dashboardUrl from '@/assets/Icons/dashboard.svg';
import wrenchUrl from '@/assets/Icons/wrench.svg';
import dottedDividerUrl from '@/assets/Images/dotted-divider.svg';

import AlertRules from './AlertRules/AlertRules';
import Dashboards from './Dashboards/Dashboards';
import DataSourceInfo from './DataSourceInfo/DataSourceInfo';
import NocDashboard from './NocDashboard/NocDashboard';
import SavedViews from './SavedViews/SavedViews';
import Services from './Services/Services';

import './Home.styles.scss';

const homeInterval = 30 * 60 * 1000;

// Toggle: render the NOC/control-center dashboard once telemetry is flowing.
// New workspaces (no data yet) keep the onboarding checklist experience.
const USE_NOC_HOME = true;

// eslint-disable-next-line sonarjs/cognitive-complexity
export default function Home(): JSX.Element {
	const { t } = useTranslation('home');
	const { user } = useAppContext();
	const { safeNavigate } = useSafeNavigate();

	const [startTime, setStartTime] = useState<number | null>(null);
	const [endTime, setEndTime] = useState<number | null>(null);

	useEffect(() => {
		const now = new Date();
		const startTime = new Date(now.getTime() - homeInterval);
		const endTime = now;

		setStartTime(startTime.getTime());
		setEndTime(endTime.getTime());
	}, []);

	// Detect Logs
	const { data: logsData, isLoading: isLogsLoading } = useGetQueryRange(
		{
			query: initialQueriesMap[DataSource.LOGS],
			graphType: PANEL_TYPES.VALUE,
			selectedTime: 'GLOBAL_TIME',
			globalSelectedInterval: DEFAULT_TIME_RANGE,
			params: {
				dataSource: DataSource.LOGS,
			},
			formatForWeb: false,
		},
		ENTITY_VERSION_V5,
		{
			queryKey: [
				REACT_QUERY_KEY.GET_QUERY_RANGE,
				DEFAULT_TIME_RANGE,
				endTime || Date.now(),
				startTime || Date.now(),
				initialQueriesMap[DataSource.LOGS],
			],
			enabled: !!startTime && !!endTime,
		},
	);

	// Detect Traces
	const { data: tracesData, isLoading: isTracesLoading } = useGetQueryRange(
		{
			query: initialQueriesMap[DataSource.TRACES],
			graphType: PANEL_TYPES.VALUE,
			selectedTime: 'GLOBAL_TIME',
			globalSelectedInterval: DEFAULT_TIME_RANGE,
			params: {
				dataSource: DataSource.TRACES,
			},
			formatForWeb: false,
		},
		ENTITY_VERSION_V5,
		{
			queryKey: [
				REACT_QUERY_KEY.GET_QUERY_RANGE,
				DEFAULT_TIME_RANGE,
				endTime || Date.now(),
				startTime || Date.now(),
				initialQueriesMap[DataSource.TRACES],
			],
			enabled: !!startTime && !!endTime,
		},
	);

	// Detect Metrics
	const { data: metricsOnboardingData } = useGetMetricsOnboardingStatus();

	const [isLogsIngestionActive, setIsLogsIngestionActive] = useState(false);
	const [isTracesIngestionActive, setIsTracesIngestionActive] = useState(false);
	const [isMetricsIngestionActive, setIsMetricsIngestionActive] =
		useState(false);

	useEffect(() => {
		if (isIngestionActive(logsData?.payload)) {
			setIsLogsIngestionActive(true);
		}
	}, [logsData]);

	useEffect(() => {
		if (isIngestionActive(tracesData?.payload)) {
			setIsTracesIngestionActive(true);
		}
	}, [tracesData]);

	useEffect(() => {
		if (metricsOnboardingData?.data?.hasMetrics) {
			setIsMetricsIngestionActive(true);
		}
	}, [metricsOnboardingData]);

	useEffect(() => {
		logEvent('Homepage: Visited', {});
	}, []);

	const isAnyIngestionActive =
		isLogsIngestionActive || isTracesIngestionActive || isMetricsIngestionActive;
	const showNocDashboard = USE_NOC_HOME && isAnyIngestionActive;

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

			<div className={`home-content${showNocDashboard ? ' home-content--noc' : ''}`}>
				{showNocDashboard ? (
					<NocDashboard />
				) : (
					<>
				<div className="home-left-content">
					<DataSourceInfo
						dataSentToSigNoz={
							isLogsIngestionActive ||
							isTracesIngestionActive ||
							isMetricsIngestionActive
						}
						isLoading={isLogsLoading || isTracesLoading}
					/>

					<div className="divider">
						<img src={dottedDividerUrl} alt="divider" />
					</div>

					<div className="active-ingestions-container">
						{isLogsIngestionActive && (
							<Card className="active-ingestion-card" size="small">
								<Card.Content>
									<div className="active-ingestion-card-content-container">
										<div className="active-ingestion-card-content">
											<div className="active-ingestion-card-content-icon">
												<Dot size={16} color={Color.BG_FOREST_500} />
											</div>

											<div className="active-ingestion-card-content-description">
												{t('ingestion_active_logs')}
											</div>
										</div>

										<div
											role="button"
											tabIndex={0}
											className="active-ingestion-card-actions"
											onClick={(e: React.MouseEvent): void => {
												// eslint-disable-next-line sonarjs/no-duplicate-string
												logEvent('Homepage: Ingestion Active Explore clicked', {
													source: 'Logs',
												});
												safeNavigate(ROUTES.LOGS_EXPLORER, {
													newTab: isModifierKeyPressed(e),
												});
											}}
											onKeyDown={(e): void => {
												if (e.key === 'Enter') {
													logEvent('Homepage: Ingestion Active Explore clicked', {
														source: 'Logs',
													});
													history.push(ROUTES.LOGS_EXPLORER);
												}
											}}
										>
											<Compass size={12} />
											{t('explore_logs')}
										</div>
									</div>
								</Card.Content>
							</Card>
						)}

						{isTracesIngestionActive && (
							<Card className="active-ingestion-card" size="small">
								<Card.Content>
									<div className="active-ingestion-card-content-container">
										<div className="active-ingestion-card-content">
											<div className="active-ingestion-card-content-icon">
												<Dot size={16} color={Color.BG_FOREST_500} />
											</div>

											<div className="active-ingestion-card-content-description">
												{t('ingestion_active_traces')}
											</div>
										</div>

										<div
											className="active-ingestion-card-actions"
											role="button"
											tabIndex={0}
											onClick={(e: React.MouseEvent): void => {
												logEvent('Homepage: Ingestion Active Explore clicked', {
													source: 'Traces',
												});
												safeNavigate(ROUTES.TRACES_EXPLORER, {
													newTab: isModifierKeyPressed(e),
												});
											}}
											onKeyDown={(e): void => {
												if (e.key === 'Enter') {
													logEvent('Homepage: Ingestion Active Explore clicked', {
														source: 'Traces',
													});
													history.push(ROUTES.TRACES_EXPLORER);
												}
											}}
										>
											<Compass size={12} />
											{t('explore_traces')}
										</div>
									</div>
								</Card.Content>
							</Card>
						)}

						{isMetricsIngestionActive && (
							<Card className="active-ingestion-card" size="small">
								<Card.Content>
									<div className="active-ingestion-card-content-container">
										<div className="active-ingestion-card-content">
											<div className="active-ingestion-card-content-icon">
												<Dot size={16} color={Color.BG_FOREST_500} />
											</div>

											<div className="active-ingestion-card-content-description">
												{t('ingestion_active_metrics')}
											</div>
										</div>

										<div
											className="active-ingestion-card-actions"
											role="button"
											tabIndex={0}
											onClick={(e: React.MouseEvent): void => {
												logEvent('Homepage: Ingestion Active Explore clicked', {
													source: 'Metrics',
												});
												safeNavigate(ROUTES.METRICS_EXPLORER, {
													newTab: isModifierKeyPressed(e),
												});
											}}
											onKeyDown={(e): void => {
												if (e.key === 'Enter') {
													logEvent('Homepage: Ingestion Active Explore clicked', {
														source: 'Metrics',
													});
													history.push(ROUTES.METRICS_EXPLORER);
												}
											}}
										>
											<Compass size={12} />
											{t('explore_metrics')}
										</div>
									</div>
								</Card.Content>
							</Card>
						)}
					</div>

					{user?.role !== USER_ROLES.VIEWER && (
						<div className="explorers-container">
							<Card className="explorer-card">
								<Card.Content>
									<div className="section-container">
										<div className="section-content">
											<div className="section-icon">
												<img
													src={wrenchUrl}
													alt="wrench"
													width={16}
													height={16}
													loading="lazy"
												/>
											</div>

											<div className="section-title">
												<div className="title">{t('explorer_title')}</div>

												<div className="description">
													{t('explorer_desc')}
												</div>
											</div>
										</div>

										<div className="section-actions">
											<Button
												variant="solid"
												color="secondary"
												className="periscope-btn secondary"
												prefix={<Wrench size={14} />}
												onClick={(e: React.MouseEvent): void => {
													logEvent('Homepage: Explore clicked', {
														source: 'Logs',
													});
													safeNavigate(ROUTES.LOGS_EXPLORER, {
														newTab: isModifierKeyPressed(e),
													});
												}}
											>
												{t('open_logs_explorer')}
											</Button>

											<Button
												variant="solid"
												color="secondary"
												className="periscope-btn secondary"
												prefix={<Wrench size={14} />}
												onClick={(e: React.MouseEvent): void => {
													logEvent('Homepage: Explore clicked', {
														source: 'Traces',
													});
													safeNavigate(ROUTES.TRACES_EXPLORER, {
														newTab: isModifierKeyPressed(e),
													});
												}}
											>
												{t('open_traces_explorer')}
											</Button>

											<Button
												variant="solid"
												color="secondary"
												className="periscope-btn secondary"
												prefix={<Wrench size={14} />}
												onClick={(e: React.MouseEvent): void => {
													logEvent('Homepage: Explore clicked', {
														source: 'Metrics',
													});
													safeNavigate(ROUTES.METRICS_EXPLORER_EXPLORER, {
														newTab: isModifierKeyPressed(e),
													});
												}}
											>
												{t('open_metrics_explorer')}
											</Button>
										</div>
									</div>
								</Card.Content>
							</Card>

							<Card className="explorer-card">
								<Card.Content>
									<div className="section-container">
										<div className="section-content">
											<div className="section-icon">
												<img src={dashboardUrl} alt="dashboard" width={16} height={16} />
											</div>

											<div className="section-title">
												<div className="title">{t('dashboard_title')}</div>

												<div className="description">
													{t('dashboard_desc')}
												</div>
											</div>
										</div>

										<div className="section-actions">
											<Button
												variant="solid"
												color="secondary"
												className="periscope-btn secondary"
												prefix={<Plus size={14} />}
												onClick={(e: React.MouseEvent): void => {
													logEvent('Homepage: Explore clicked', {
														source: 'Dashboards',
													});
													safeNavigate(ROUTES.ALL_DASHBOARD, {
														newTab: isModifierKeyPressed(e),
													});
												}}
											>
												{t('create_dashboard')}
											</Button>
										</div>
									</div>
								</Card.Content>
							</Card>

							<Card className="explorer-card">
								<Card.Content>
									<div className="section-container">
										<div className="section-content">
											<div className="section-icon">
												<img
													src={crackerUrl}
													alt="cracker"
													width={16}
													height={16}
													loading="lazy"
												/>
											</div>

											<div className="section-title">
												<div className="title">{t('alert_title')}</div>

												<div className="description">
													{t('alert_desc')}
												</div>
											</div>
										</div>

										<div className="section-actions">
											<Button
												variant="solid"
												color="secondary"
												className="periscope-btn secondary"
												prefix={<Plus size={14} />}
												onClick={(e: React.MouseEvent): void => {
													logEvent('Homepage: Explore clicked', {
														source: 'Alerts',
													});
													safeNavigate(ROUTES.ALERTS_NEW, {
														newTab: isModifierKeyPressed(e),
													});
												}}
											>
												{t('create_alert')}
											</Button>
										</div>
									</div>
								</Card.Content>
							</Card>
						</div>
					)}

					{(isLogsIngestionActive ||
						isTracesIngestionActive ||
						isMetricsIngestionActive) && (
						<>
							<AlertRules />
							<Dashboards />
						</>
					)}
				</div>
				<div className="home-right-content">
					{(isLogsIngestionActive ||
						isTracesIngestionActive ||
						isMetricsIngestionActive) && (
						<>
							<Services />
							<SavedViews />
						</>
					)}
				</div>
					</>
				)}
			</div>
		</div>
	);
}
