import GridCardGraph from 'container/GridCardLayout/GridCard';
import { Pin, Plus, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { NocPinnedSlot } from '../types';
import { PIN_CAP } from '../utils/pinnedPanels';

export interface PinnedPanelsProps {
	slots: NocPinnedSlot[];
	onUnpin: (widgetId: string) => void;
	onOpenPicker: () => void;
}

const noop = (): void => {};

export default function PinnedPanels({
	slots,
	onUnpin,
	onOpenPicker,
}: PinnedPanelsProps): JSX.Element {
	const { t } = useTranslation('home');

	return (
		<div className="noc-c2-pins">
			{slots.map((slot) => (
				<div className="noc-c2-pin-card" key={slot.ref.widgetId}>
					<div className="noc-c2-pin-head">
						<Pin size={11} className="noc-c2-pin-ico" />
						<span className="noc-c2-pin-src">{slot.dashboardTitle}</span>
						<button
							type="button"
							className="noc-c2-pin-unpin"
							aria-label={t('noc_c2_pin_unpin').toString()}
							title={t('noc_c2_pin_unpin').toString()}
							onClick={(): void => onUnpin(slot.ref.widgetId)}
						>
							<X size={12} />
						</button>
					</div>
					<div className="noc-c2-pin-body">
						{slot.widget ? (
							<GridCardGraph
								widget={slot.widget}
								isQueryEnabled
								version={slot.version}
								headerMenuList={[]}
								onDragSelect={noop}
							/>
						) : (
							<div className="noc-c2-pin-missing">{t('noc_c2_pin_missing')}</div>
						)}
					</div>
				</div>
			))}
			{slots.length < PIN_CAP ? (
				<button type="button" className="noc-c2-pin-add" onClick={onOpenPicker}>
					<Plus size={14} />
					<span>{t('noc_c2_pin_add')}</span>
					<span className="noc-c2-pin-add-hint">{t('noc_c2_pin_empty_hint')}</span>
				</button>
			) : null}
		</div>
	);
}
