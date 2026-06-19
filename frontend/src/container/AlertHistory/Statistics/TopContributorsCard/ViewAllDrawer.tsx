import { Color } from '@signozhq/design-tokens';
import { Drawer } from 'antd';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useTranslation } from 'react-i18next';
import { AlertRuleStats, AlertRuleTopContributors } from 'types/api/alerts/def';

import TopContributorsRows from './TopContributorsRows';

function ViewAllDrawer({
	isViewAllVisible,
	toggleViewAllDrawer,
	totalCurrentTriggers,
	topContributorsData,
}: {
	isViewAllVisible: boolean;
	toggleViewAllDrawer: () => void;
	topContributorsData: AlertRuleTopContributors[];
	totalCurrentTriggers: AlertRuleStats['totalCurrentTriggers'];
}): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { t } = useTranslation('alerts');
	return (
		<Drawer
			open={isViewAllVisible}
			destroyOnClose
			onClose={toggleViewAllDrawer}
			placement="right"
			width="50%"
			className="view-all-drawer"
			style={{
				overscrollBehavior: 'contain',
				background: isDarkMode ? Color.BG_INK_400 : Color.BG_VANILLA_100,
			}}
			title={t('hist_viewing_all_contributors')}
		>
			<div className="top-contributors-card--view-all">
				<div className="top-contributors-card__content">
					<TopContributorsRows
						topContributors={topContributorsData}
						totalCurrentTriggers={totalCurrentTriggers}
					/>
				</div>
			</div>
		</Drawer>
	);
}

export default ViewAllDrawer;
