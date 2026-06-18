import { memo, MouseEventHandler } from 'react';
import { useTranslation } from 'react-i18next';
import { LinkOutlined } from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import { TextSelect } from 'lucide-react';

import './LogLinesActionButtons.styles.scss';

export interface LogLinesActionButtonsProps {
	handleShowContext: MouseEventHandler<HTMLElement>;
	onLogCopy: MouseEventHandler<HTMLElement>;
	customClassName?: string;
}

function LogLinesActionButtons({
	handleShowContext,
	onLogCopy,
	customClassName = '',
}: LogLinesActionButtonsProps): JSX.Element {
	const { t } = useTranslation(['logs']);
	return (
		<div className={`log-line-action-buttons ${customClassName}`}>
			<Tooltip title={t('logs:show_in_context')}>
				<Button
					size="small"
					icon={<TextSelect size={14} />}
					className="show-context-btn"
					onClick={handleShowContext}
				/>
			</Tooltip>
			<Tooltip title={t('logs:copy_link')}>
				<Button
					size="small"
					icon={<LinkOutlined size={14} />}
					onClick={onLogCopy}
					className="copy-log-btn"
				/>
			</Tooltip>
		</div>
	);
}

LogLinesActionButtons.defaultProps = {
	customClassName: '',
};

export default memo(LogLinesActionButtons);
