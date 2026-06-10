import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Button } from 'antd';
import cx from 'classnames';
import { ORDERBY_FILTERS } from 'container/QueryBuilder/filters/OrderByFilter/config';
import { ArrowDown, ArrowUp, Ban } from 'lucide-react';

import './ShowButton.styles.scss';

interface ShowButtonProps {
	isLoading: boolean;
	isDisabled: boolean;
	order: string;
	onClick: () => void;
}

function ShowButton({
	isLoading,
	isDisabled,
	order,
	onClick,
}: ShowButtonProps): JSX.Element {
	const { t } = useTranslation(['logs']);
	const getIcons = (): JSX.Element => {
		if (order === ORDERBY_FILTERS.ASC) {
			return isDisabled ? (
				<Ban size={14} style={{ color: Color.BG_VANILLA_400 }} />
			) : (
				<ArrowUp size={14} />
			);
		}
		return isDisabled ? (
			<Ban size={14} style={{ color: Color.BG_VANILLA_400 }} />
		) : (
			<ArrowDown size={14} />
		);
	};

	return (
		<Button
			disabled={isLoading || isDisabled}
			loading={isLoading}
			onClick={onClick}
			icon={getIcons()}
			className={cx(
				'show-more-button periscope-btn',
				order === ORDERBY_FILTERS.ASC ? 'up' : 'down',
				isDisabled && 'disabled',
			)}
		>
			{t('logs:load_more')}
		</Button>
	);
}

export default ShowButton;
