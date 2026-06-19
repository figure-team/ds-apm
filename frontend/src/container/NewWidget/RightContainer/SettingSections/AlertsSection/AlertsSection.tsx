import { Typography } from 'antd';
import { ConciergeBell, Plus, SquareArrowOutUpRight } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import './AlertsSection.styles.scss';

interface AlertsSectionProps {
	onCreateAlertsHandler: () => void;
}

export default function AlertsSection({
	onCreateAlertsHandler,
}: AlertsSectionProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<section className="alerts-section" onClick={onCreateAlertsHandler}>
			<div className="alerts-section__left">
				<ConciergeBell size={14} className="alerts-section__bell-icon" />
				<Typography.Text className="alerts-section__text">{t('section_alerts')}</Typography.Text>
				<SquareArrowOutUpRight size={10} className="info-icon" />
			</div>
			<Plus size={14} className="alerts-section__plus-icon" />
		</section>
	);
}
