import { useTranslation } from 'react-i18next';
import { Space, Typography } from 'antd';
import HeaderRightSection from 'components/HeaderRightSection/HeaderRightSection';
import ListOfAllDashboard from 'container/ListOfDashboard';
import { LayoutGrid } from 'lucide-react';

import './DashboardsListPage.styles.scss';

function DashboardsListPage(): JSX.Element {
	const { t } = useTranslation(['dashboard']);
	return (
		<Space
			direction="vertical"
			size="middle"
			style={{ width: '100%' }}
			className="dashboard-list-page"
		>
			<div className="dashboard-header">
				<div className="dashboard-header-left">
					<LayoutGrid size={14} className="icon" />
					<Typography.Text className="text">
						{t('dashboards_title')}
					</Typography.Text>
				</div>

				<HeaderRightSection
					enableAnnouncements={false}
					enableShare
					enableFeedback
				/>
			</div>
			<ListOfAllDashboard />
		</Space>
	);
}

export default DashboardsListPage;
