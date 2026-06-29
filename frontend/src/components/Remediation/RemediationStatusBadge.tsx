import { useTranslation } from 'react-i18next';

interface Props {
	status: string;
}

function RemediationStatusBadge({ status }: Props): JSX.Element {
	const { t } = useTranslation('alerts');
	return (
		<span className={`remediation-card__badge remediation-card__badge--${status}`}>
			{t(`remediation_status_${status}`)}
		</span>
	);
}

export default RemediationStatusBadge;
