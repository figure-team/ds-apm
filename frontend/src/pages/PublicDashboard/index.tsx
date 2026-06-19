﻿import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import { Typography } from 'antd';
import { useGetPublicDashboardData } from 'hooks/dashboard/useGetPublicDashboardData';
import { FrownIcon } from 'lucide-react';

import signozBrandLogoUrl from '@/assets/Logos/signoz-brand-logo.svg';

import PublicDashboardContainer from '../../container/PublicDashboardContainer';

import './PublicDashboard.styles.scss';

function PublicDashboardPage(): JSX.Element {
	// read the dashboard id from the url
	const { t } = useTranslation('dashboard');

	const { dashboardId } = useParams<{ dashboardId: string }>();

	const {
		data: publicDashboardData,
		isLoading: isLoadingPublicDashboardData,
		isFetching: isFetchingPublicDashboardData,
		isError: isErrorPublicDashboardData,
	} = useGetPublicDashboardData(dashboardId || '');

	const isLoading =
		isLoadingPublicDashboardData || isFetchingPublicDashboardData;

	const isError = isErrorPublicDashboardData;

	return (
		<div className="public-dashboard-page">
			{publicDashboardData && (
				<PublicDashboardContainer
					publicDashboardId={dashboardId}
					publicDashboardData={publicDashboardData}
				/>
			)}

			{isError && !isLoading && (
				<div className="public-dashboard-error-container">
					<div className="perilin-bg" />

					<div className="public-dashboard-error-content-header">
						<div className="brand">
							<img src={signozBrandLogoUrl} alt={t('public_dashboard_brand')} className="brand-logo" />

							<Typography.Title level={2} className="brand-title">
								{t('public_dashboard_brand')}
							</Typography.Title>
						</div>

						<div className="brand-tagline">
							<Typography.Text>
								{t('public_dashboard_tagline')}
							</Typography.Text>
						</div>
					</div>

					<div className="public-dashboard-error-content">
						<Typography.Title
							level={4}
							className="public-dashboard-error-message-icon"
						>
							<FrownIcon size={36} />
						</Typography.Title>
						<Typography.Title level={4} className="public-dashboard-error-message">
							{t('public_dashboard_not_found')}
						</Typography.Title>
						<Typography.Text className="public-dashboard-error-message-description">
							{t('public_dashboard_contact_owner')}
						</Typography.Text>
					</div>
				</div>
			)}
		</div>
	);
}

export default PublicDashboardPage;
