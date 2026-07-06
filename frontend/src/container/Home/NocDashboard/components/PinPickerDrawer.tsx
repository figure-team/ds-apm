import { Checkbox, Collapse, Drawer } from 'antd';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Dashboard } from 'types/api/dashboard/getAll';

import { NocPinnedRef } from '../types';
import { listPinnableWidgets, PIN_CAP } from '../utils/pinnedPanels';

export interface PinPickerDrawerProps {
	open: boolean;
	onClose: () => void;
	dashboards: Dashboard[];
	refs: NocPinnedRef[];
	onPin: (ref: NocPinnedRef) => void;
	onUnpin: (widgetId: string) => void;
}

export default function PinPickerDrawer({
	open,
	onClose,
	dashboards,
	refs,
	onPin,
	onUnpin,
}: PinPickerDrawerProps): JSX.Element {
	const { t } = useTranslation('home');
	const pinnedIds = useMemo(() => new Set(refs.map((r) => r.widgetId)), [refs]);
	const full = refs.length >= PIN_CAP;

	const items = useMemo(
		() =>
			dashboards
				.map((d) => ({ dashboard: d, widgets: listPinnableWidgets(d) }))
				.filter(({ widgets }) => widgets.length > 0)
				.map(({ dashboard, widgets }) => ({
					key: dashboard.id,
					label: dashboard.data.title,
					children: (
						<div className="noc-c2-pin-picker-list">
							{widgets.map((w) => {
								const checked = pinnedIds.has(w.id);
								return (
									<label className="noc-c2-pin-picker-row" key={w.id}>
										<Checkbox
											checked={checked}
											disabled={!checked && full}
											aria-label={typeof w.title === 'string' ? w.title : ''}
											onChange={(e): void => {
												if (e.target.checked) {
													onPin({ dashboardId: dashboard.id, widgetId: w.id });
												} else {
													onUnpin(w.id);
												}
											}}
										/>
										<span className="noc-c2-pin-picker-title">{w.title}</span>
										<span className="noc-c2-pin-picker-type">{w.panelTypes}</span>
									</label>
								);
							})}
						</div>
					),
				})),
		[dashboards, pinnedIds, full, onPin, onUnpin],
	);

	return (
		<Drawer
			title={t('noc_c2_pin_drawer_title')}
			open={open}
			onClose={onClose}
			width={420}
			rootClassName="noc-c2-pin-drawer"
		>
			<div className="noc-c2-pin-drawer-hint">
				{t('noc_c2_pin_drawer_hint', { max: PIN_CAP })}
			</div>
			{items.length === 0 ? (
				<div className="noc-c2-pin-drawer-empty">
					{t('noc_c2_pin_drawer_empty')}
				</div>
			) : (
				<Collapse items={items} ghost />
			)}
		</Drawer>
	);
}
