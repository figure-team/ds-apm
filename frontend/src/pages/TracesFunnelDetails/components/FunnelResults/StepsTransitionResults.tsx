import { useMemo, useState } from 'react';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';
import SignozRadioGroup from 'components/SignozRadioGroup/SignozRadioGroup';
import { useFunnelContext } from 'pages/TracesFunnels/FunnelContext';

import StepsTransitionMetrics from './StepsTransitionMetrics';
import TopSlowestTraces from './TopSlowestTraces';
import TopTracesWithErrors from './TopTracesWithErrors';

import './StepsTransitionResults.styles.scss';

export interface StepTransition {
	value: string;
	label: string;
}

function generateStepTransitions(
	stepsCount: number,
	t: TFunction,
): StepTransition[] {
	return Array.from({ length: stepsCount - 1 }, (_, index) => ({
		value: `${index + 1}_to_${index + 2}`,
		label: t('funnels.step_transition_label', {
			from: index + 1,
			to: index + 2,
		}).toString(),
	}));
}

function StepsTransitionResults(): JSX.Element {
	const { t } = useTranslation('trace');
	const { steps, funnelId } = useFunnelContext();
	const stepTransitions = generateStepTransitions(steps.length, t);
	const [selectedTransition, setSelectedTransition] = useState<string>(
		stepTransitions[0]?.value || '',
	);

	const [stepAOrder, stepBOrder] = useMemo(() => {
		const [a, b] = selectedTransition.split('_to_');
		return [parseInt(a, 10), parseInt(b, 10)];
	}, [selectedTransition]);

	return (
		<div className="steps-transition-results">
			<div className="steps-transition-results__steps-selector">
				<SignozRadioGroup
					value={selectedTransition}
					options={stepTransitions}
					onChange={(e): void => setSelectedTransition(e.target.value)}
				/>
			</div>
			<div className="steps-transition-results__results">
				<StepsTransitionMetrics
					selectedTransition={selectedTransition}
					transitions={stepTransitions}
					startStep={stepAOrder}
					endStep={stepBOrder}
				/>
				<TopSlowestTraces
					funnelId={funnelId}
					stepAOrder={stepAOrder}
					stepBOrder={stepBOrder}
					steps={steps}
				/>
				<TopTracesWithErrors
					funnelId={funnelId}
					stepAOrder={stepAOrder}
					stepBOrder={stepBOrder}
					steps={steps}
				/>
			</div>
		</div>
	);
}

export default StepsTransitionResults;
