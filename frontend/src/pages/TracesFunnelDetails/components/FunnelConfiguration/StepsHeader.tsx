import { useTranslation } from 'react-i18next';
import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';

import './StepsHeader.styles.scss';

function StepsHeader(): JSX.Element {
	const { t } = useTranslation('trace');
	return (
		<div className="steps-header">
			<div className="steps-header__label">{t('funnels.funnel_steps')}</div>
			<div className="steps-header__time-range">
				<DateTimeSelectionV2
					showAutoRefresh={false}
					showRefreshText={false}
					hideShareModal
					showRecentlyUsed={false}
				/>
			</div>
		</div>
	);
}

export default StepsHeader;
