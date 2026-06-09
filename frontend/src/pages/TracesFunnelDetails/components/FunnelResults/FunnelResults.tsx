import { useIsMutating } from 'react-query';
import { useTranslation } from 'react-i18next';
import Spinner from 'components/Spinner';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { useFunnelContext } from 'pages/TracesFunnels/FunnelContext';

import EmptyFunnelResults from './EmptyFunnelResults';
import FunnelGraph from './FunnelGraph';
import OverallMetrics from './OverallMetrics';
import StepsTransitionResults from './StepsTransitionResults';

import './FunnelResults.styles.scss';

function FunnelResults(): JSX.Element {
	const { t } = useTranslation('trace');
	const {
		validTracesCount,
		isValidateStepsLoading,
		hasIncompleteStepFields,
		hasAllEmptyStepFields,
		funnelId,
	} = useFunnelContext();

	const isFunnelUpdateMutating = useIsMutating([
		REACT_QUERY_KEY.UPDATE_FUNNEL_STEPS,
		funnelId,
	]);

	if (hasAllEmptyStepFields) {
		return <EmptyFunnelResults />;
	}

	if (hasIncompleteStepFields) {
		return (
			<EmptyFunnelResults
				title={t('funnels.missing_names_title')}
				description={t('funnels.missing_names_desc')}
			/>
		);
	}

	if (isValidateStepsLoading || isFunnelUpdateMutating) {
		return <Spinner size="large" />;
	}

	if (validTracesCount === 0) {
		return (
			<EmptyFunnelResults
				title={t('funnels.no_traces_title')}
				description={t('funnels.no_traces_desc')}
			/>
		);
	}

	return (
		<div className="funnel-results">
			<OverallMetrics />
			<FunnelGraph />
			<StepsTransitionResults />
		</div>
	);
}

export default FunnelResults;
