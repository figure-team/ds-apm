import { useTranslation } from 'react-i18next';
import { useFunnelErrorTraces } from 'hooks/TracesFunnels/useFunnels';
import { FunnelStepData } from 'types/api/traceFunnels';

import FunnelTopTracesTable from './FunnelTopTracesTable';

interface TopTracesWithErrorsProps {
	funnelId: string;
	stepAOrder: number;
	stepBOrder: number;
	steps: FunnelStepData[];
}

function TopTracesWithErrors(props: TopTracesWithErrorsProps): JSX.Element {
	const { t } = useTranslation('trace');
	return (
		<FunnelTopTracesTable
			{...props}
			title={t('funnels.error_traces_title')}
			tooltip={t('funnels.error_traces_tooltip')}
			useQueryHook={useFunnelErrorTraces}
		/>
	);
}

export default TopTracesWithErrors;
