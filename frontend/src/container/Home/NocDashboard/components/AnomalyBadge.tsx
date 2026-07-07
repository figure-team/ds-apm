import { Popover } from 'antd';
import { Siren } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { popupContainer } from 'utils/selectPopupContainer';

import { NocServiceRow } from '../types';
import WatchCards from './WatchCards';

export interface AnomalyBadgeProps {
	/** 이상 서비스 목록(상한 적용분) — 비어 있으면 배지 자체를 렌더하지 않음 */
	services: NocServiceRow[];
	/** 배지에 표시할 전체 이상 서비스 수 (critical + warning) */
	count: number;
	overflowCount?: number;
}

export default function AnomalyBadge({
	services,
	count,
	overflowCount = 0,
}: AnomalyBadgeProps): JSX.Element | null {
	const { t } = useTranslation('home');
	const [open, setOpen] = useState(false);

	if (services.length === 0) {
		return null;
	}

	return (
		<Popover
			open={open}
			onOpenChange={setOpen}
			trigger="click"
			placement="bottomRight"
			arrow
			getPopupContainer={popupContainer}
			rootClassName="noc-c2-anom-popover"
			content={
				<div className="noc-c2-anom-pop-body">
					<WatchCards
						services={services}
						mode="anomaly"
						overflowCount={overflowCount}
					/>
				</div>
			}
		>
			<button type="button" className="noc-c2-anom-badge">
				<Siren size={12} />
				<span>{t('noc_c2_anom_badge', { count })}</span>
				<span className="noc-c2-anom-caret">{open ? '▴' : '▾'}</span>
			</button>
		</Popover>
	);
}
