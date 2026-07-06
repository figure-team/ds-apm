import { Popover } from 'antd';
import { Server } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { popupContainer } from 'utils/selectPopupContainer';

import InfraPanel, { InfraPanelProps } from './InfraPanel';

function toneClass(hosts: InfraPanelProps['hosts']): string {
	if (hosts.some((h) => h.health === 'critical')) return 'noc-c2-infra-crit';
	if (hosts.some((h) => h.health === 'warning')) return 'noc-c2-infra-warn';
	return 'noc-c2-infra-calm';
}

export default function InfraBadge({
	hosts,
	isLoading,
	isError,
}: InfraPanelProps): JSX.Element {
	const { t } = useTranslation('home');
	const [open, setOpen] = useState(false);
	const attention = useMemo(
		() => hosts.filter((h) => h.health !== 'healthy').length,
		[hosts],
	);

	return (
		<Popover
			open={open}
			onOpenChange={setOpen}
			trigger="click"
			placement="bottomRight"
			arrow
			getPopupContainer={popupContainer}
			rootClassName="noc-c2-infra-popover"
			content={
				<div className="noc-c2-infra-pop-body">
					<InfraPanel hosts={hosts} isLoading={isLoading} isError={isError} />
				</div>
			}
		>
			<button type="button" className={`noc-c2-infra-badge ${toneClass(hosts)}`}>
				<Server size={12} />
				<span>{t('noc_c2_infra_badge', { count: hosts.length })}</span>
				{attention > 0 ? (
					<span className="noc-c2-infra-attn">
						{t('noc_c2_infra_attn', { count: attention })}
					</span>
				) : null}
				<span className="noc-c2-infra-caret">{open ? '▴' : '▾'}</span>
			</button>
		</Popover>
	);
}
