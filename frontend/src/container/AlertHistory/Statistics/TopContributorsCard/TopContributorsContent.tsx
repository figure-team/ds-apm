import { useTranslation } from 'react-i18next';

import TopContributorsRows from './TopContributorsRows';
import { TopContributorsCardProps } from './types';

function TopContributorsContent({
	topContributorsData,
	totalCurrentTriggers,
}: TopContributorsCardProps): JSX.Element {
	const { t } = useTranslation('alerts');
	const isEmpty = !topContributorsData.length;

	if (isEmpty) {
		return (
			<div className="empty-content">
				<div className="empty-content__icon">ℹ️</div>
				<div className="empty-content__text">
					{t('hist_top_contributors_desc')}
				</div>
			</div>
		);
	}

	return (
		<div className="top-contributors-card__content">
			<TopContributorsRows
				topContributors={topContributorsData.slice(0, 3)}
				totalCurrentTriggers={totalCurrentTriggers}
			/>
		</div>
	);
}

export default TopContributorsContent;
