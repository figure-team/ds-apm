import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Tooltip } from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import { Info } from 'lucide-react';

import './MQCommon.styles.scss';

export function ComingSoon(): JSX.Element {
	const { t } = useTranslation('messagingQueues');
	return (
		<Tooltip
			title={
				<div>
					{t('join_slack')}{' '}
					<a
						href="https://signoz.io/slack"
						rel="noopener noreferrer"
						target="_blank"
						onClick={(e): void => e.stopPropagation()}
					>
						{t('signoz_community')}
					</a>
				</div>
			}
			placement="top"
			overlayClassName="tooltip-overlay"
		>
			<div className="coming-soon">
				<div className="coming-soon__text">{t('coming_soon')}</div>
				<div className="coming-soon__icon">
					<Info size={10} color={Color.BG_SIENNA_400} />
				</div>
			</div>
		</Tooltip>
	);
}

export function SelectMaxTagPlaceholder(
	omittedValues: Partial<DefaultOptionType>[],
): JSX.Element {
	return (
		<Tooltip title={omittedValues.map(({ value }) => value).join(', ')}>
			<span>+ {omittedValues.length} </span>
		</Tooltip>
	);
}

export function SelectLabelWithComingSoon({
	label,
}: {
	label: string;
}): JSX.Element {
	return (
		<div className="select-label-with-coming-soon">
			{label} <ComingSoon />
		</div>
	);
}
