import { useTranslation } from 'react-i18next';

export interface TrendLegendItem {
	name: string;
	color: string;
	missing?: boolean;
	hidden: boolean;
}

export interface TrendLegendProps {
	items: TrendLegendItem[];
	hovered: string | null;
	onHover: (name: string | null) => void;
	onToggle: (name: string) => void;
}

export default function TrendLegend({
	items,
	hovered,
	onHover,
	onToggle,
}: TrendLegendProps): JSX.Element {
	const { t } = useTranslation('home');
	return (
		<div className="noc-c2-trend-legend">
			{items.map((it) => (
				<button
					key={it.name}
					type="button"
					className={`noc-c2-legend-item${it.missing ? ' missing' : ''}${
						it.hidden ? ' hidden' : ''
					}${hovered && hovered !== it.name ? ' dim' : ''}`}
					aria-pressed={it.hidden}
					onMouseEnter={(): void => onHover(it.name)}
					onMouseLeave={(): void => onHover(null)}
					onClick={(): void => onToggle(it.name)}
				>
					<span
						className="noc-c2-legend-swatch"
						style={{ background: it.missing ? 'var(--noc-c-sec)' : it.color }}
					/>
					<span className="noc-c2-legend-name">{it.name}</span>
					{it.missing ? (
						<span className="noc-c2-legend-nodata">{t('noc_c2_series_nodata')}</span>
					) : null}
				</button>
			))}
		</div>
	);
}
