import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux'; // old code, TODO: fix this correctly
import { Link } from 'react-router-dom';
import { Button, Select, Skeleton, Table } from 'antd';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import { useQueryService } from 'hooks/useQueryService';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { ArrowRight } from 'lucide-react';
import Card from 'periscope/components/Card/Card';
import { AppState } from 'store/reducers';
import { ServicesList } from 'types/api/metrics/getService';
import { GlobalReducer } from 'types/reducer/globalTime';
import { isModifierKeyPressed } from 'utils/app';

import { columns, TIME_PICKER_OPTIONS } from './constants';
import ServicesEmptyState from './ServicesEmptyState';

const homeInterval = 30 * 60 * 1000;

export default function ServiceTraces({
	onUpdateChecklistDoneItem,
	loadingUserPreferences,
}: {
	onUpdateChecklistDoneItem: (itemKey: string) => void;
	loadingUserPreferences: boolean;
}): JSX.Element {
	const { selectedTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);
	const { t } = useTranslation('home');

	const now = new Date().getTime();
	const [timeRange, setTimeRange] = useState({
		startTime: now - homeInterval,
		endTime: now,
		selectedInterval: homeInterval,
	});

	const { safeNavigate } = useSafeNavigate();

	// Fetch Services
	const {
		data: services,
		isLoading: isServicesLoading,
		isFetching: isServicesFetching,
		isError: isServicesError,
	} = useQueryService({
		minTime: timeRange.startTime * 1e6,
		maxTime: timeRange.endTime * 1e6,
		selectedTime,
		selectedTags: [],
		options: {
			enabled: true,
		},
	});

	const sortedServices = useMemo(
		() => services?.sort((a, b) => b.p99 - a.p99) || [],
		[services],
	);

	const servicesExist = useMemo(
		() => sortedServices.length > 0,
		[sortedServices],
	);
	const top5Services = useMemo(
		() => sortedServices.slice(0, 5),
		[sortedServices],
	);

	useEffect(() => {
		if (servicesExist && !loadingUserPreferences) {
			onUpdateChecklistDoneItem('SETUP_SERVICES');
		}
	}, [servicesExist, onUpdateChecklistDoneItem, loadingUserPreferences]);

	const handleTimeIntervalChange = useCallback((value: number): void => {
		const now = new Date();

		const timeInterval = TIME_PICKER_OPTIONS.find(
			(option) => option.value === value,
		);

		logEvent('Homepage: Services time interval updated', {
			updatedTimeInterval: timeInterval?.label,
		});

		setTimeRange({
			startTime: now.getTime() - value,
			endTime: now.getTime(),
			selectedInterval: value,
		});
	}, []);

	const renderDashboardsList = useCallback(
		() => (
			<div className="services-list-container home-data-item-container traces-services-list">
				<div className="services-list">
					<Table<ServicesList>
						columns={columns}
						dataSource={top5Services}
						pagination={false}
						className="services-table"
						onRow={(record: ServicesList): Record<string, unknown> => ({
							onClick: (event: React.MouseEvent): void => {
								logEvent('Homepage: Service clicked', {
									serviceName: record.serviceName,
								});

								safeNavigate(`${ROUTES.APPLICATION}/${record.serviceName}`, {
									newTab: isModifierKeyPressed(event),
								});
							},
						})}
					/>
				</div>
			</div>
		),
		[top5Services, safeNavigate],
	);

	if (isServicesLoading || isServicesFetching) {
		return (
			<Card className="dashboards-list-card home-data-card loading-card">
				<Card.Content>
					<Skeleton active />
				</Card.Content>
			</Card>
		);
	}

	if (isServicesError) {
		return (
			<Card className="dashboards-list-card home-data-card">
				<Card.Content>
					<Skeleton active />
				</Card.Content>
			</Card>
		);
	}

	return (
		<Card className="dashboards-list-card home-data-card">
			{servicesExist && (
				<Card.Header>
					<div className="services-header home-data-card-header">
						{t('services_header')}
						<div className="services-header-actions">
							<Select
								value={timeRange.selectedInterval}
								onChange={handleTimeIntervalChange}
								options={TIME_PICKER_OPTIONS}
								className="services-header-select"
							/>
						</div>
					</div>
				</Card.Header>
			)}
			<Card.Content>
				{servicesExist ? (
					renderDashboardsList()
				) : (
					<ServicesEmptyState source="Service Traces" />
				)}
			</Card.Content>

			{servicesExist && (
				<Card.Footer>
					<div className="services-footer home-data-card-footer">
						<Link to="/services">
							<Button
								type="link"
								className="periscope-btn link learn-more-link"
								onClick={(): void => {
									logEvent('Homepage: All Services clicked', {});
								}}
							>
								{t('all_services')} <ArrowRight size={12} />
							</Button>
						</Link>
					</div>
				</Card.Footer>
			)}
		</Card>
	);
}
