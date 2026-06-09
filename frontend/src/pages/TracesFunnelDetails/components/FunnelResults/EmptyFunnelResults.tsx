import LearnMore from 'components/LearnMore/LearnMore';
import { useTranslation } from 'react-i18next';

import emptyFunnelIconUrl from '@/assets/Icons/empty-funnel-icon.svg';

import './EmptyFunnelResults.styles.scss';

function EmptyFunnelResults({
	title,
	description,
}: {
	title?: string;
	description?: string;
}): JSX.Element {
	const { t } = useTranslation('trace');
	return (
		<div className="funnel-results funnel-results--empty">
			<div className="empty-funnel-results">
				<div className="empty-funnel-results__icon">
					<img src={emptyFunnelIconUrl} alt="Empty funnel results" />
				</div>
				<div className="empty-funnel-results__title">
					{title ?? t('funnels.empty_results_title')}
				</div>
				<div className="empty-funnel-results__description">
					{description ?? t('funnels.empty_results_desc')}
				</div>
				<div className="empty-funnel-results__learn-more">
					<LearnMore url="https://signoz.io/blog/tracing-funnels-observability-distributed-systems/" />
				</div>
			</div>
		</div>
	);
}

EmptyFunnelResults.defaultProps = {
	title: undefined,
	description: undefined,
};

export default EmptyFunnelResults;
