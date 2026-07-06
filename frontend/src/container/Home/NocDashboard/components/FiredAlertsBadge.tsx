import ROUTES from 'constants/routes';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { Bell } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export interface FiredAlertsBadgeProps {
	count: number;
}

export default function FiredAlertsBadge({
	count,
}: FiredAlertsBadgeProps): JSX.Element {
	const { t } = useTranslation('home');
	const { safeNavigate } = useSafeNavigate();

	return (
		<button
			type="button"
			className={`noc-c2-fired${count === 0 ? ' noc-c2-fired-quiet' : ''}`}
			onClick={(): void => safeNavigate(ROUTES.LIST_ALL_ALERT)}
		>
			<Bell size={12} />
			<span>{t('noc_c2_fired', { count })}</span>
			<span className="noc-c2-fired-arrow">→</span>
		</button>
	);
}
