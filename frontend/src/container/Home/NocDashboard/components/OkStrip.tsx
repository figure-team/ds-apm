import ROUTES from 'constants/routes';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { useTranslation } from 'react-i18next';

export interface OkStripProps {
	names: string[];
	maxChips?: number;
}

export default function OkStrip({
	names,
	maxChips = 8,
}: OkStripProps): JSX.Element {
	const { t } = useTranslation('home');
	const { safeNavigate } = useSafeNavigate();
	const shown = names.slice(0, maxChips);
	const overflow = names.length - shown.length;

	return (
		<div className="noc-ok-strip noc-c2-okstrip">
			<span className="noc-c2-okstrip-label">
				{t('noc_c2_ok_label', { count: names.length })}
			</span>
			<div className="noc-c2-okstrip-chips">
				{shown.map((n) => (
					<button
						key={n}
						type="button"
						className="noc-c2-chip"
						onClick={(): void => safeNavigate(`${ROUTES.APPLICATION}/${n}`)}
					>
						{n}
					</button>
				))}
				{overflow > 0 ? (
					<span className="noc-c2-chip noc-c2-chip-more">+{overflow}</span>
				) : null}
			</div>
		</div>
	);
}
