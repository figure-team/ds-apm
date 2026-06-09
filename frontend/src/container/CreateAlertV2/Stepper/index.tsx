import { useTranslation } from 'react-i18next';

import './styles.scss';

interface StepperProps {
	stepNumber: number;
	label: string;
	required?: boolean;
}

function Stepper({ stepNumber, label, required }: StepperProps): JSX.Element {
	const { t } = useTranslation(['alerts']);
	return (
		<div className="stepper-container">
			<div className="step-number">{stepNumber}</div>
			<div className="step-label">{label}</div>
			{required && (
				<span className="alert-header__required-badge">
					{t('v2_alert_name_required')}
				</span>
			)}
			<div className="dotted-line" />
		</div>
	);
}

export default Stepper;
