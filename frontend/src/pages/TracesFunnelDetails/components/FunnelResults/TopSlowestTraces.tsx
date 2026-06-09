import { useTranslation } from 'react-i18next';
import { useFunnelSlowTraces } from 'hooks/TracesFunnels/useFunnels';
import { FunnelStepData } from 'types/api/traceFunnels';

import FunnelTopTracesTable from './FunnelTopTracesTable';

interface TopSlowestTracesProps {
	funnelId: string;
	stepAOrder: number;
	stepBOrder: number;
	steps: FunnelStepData[];
}

function TopSlowestTraces(props: TopSlowestTracesProps): JSX.Element {
	const { t } = useTranslation('trace');
	return (
		<FunnelTopTracesTable
			{...props}
			title={t('funnels.slowest_traces_title')}
			tooltip={t('funnels.slowest_traces_tooltip')}
			useQueryHook={useFunnelSlowTraces}
		/>
	);
}

export default TopSlowestTraces;
