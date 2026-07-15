import { ReactNode, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useCopyToClipboard } from 'react-use';
import { Popover } from 'antd';
import { useNotifications } from 'hooks/useNotifications';

function CopyClipboardHOC({
	entityKey,
	textToCopy,
	tooltipText = 'Copy to clipboard',
	children,
}: CopyClipboardHOCProps): JSX.Element {
	const [value, setCopy] = useCopyToClipboard();
	const { notifications } = useNotifications();
	const { t } = useTranslation('logs');
	useEffect(() => {
		if (value.value) {
			const key = entityKey || '';

			const notificationMessage = t('attribute_copied_to_clipboard', { key });

			notifications.success({
				message: notificationMessage,
				key: notificationMessage,
			});
		}
	}, [value, notifications, entityKey, t]);

	const onClick = useCallback((): void => {
		setCopy(textToCopy);
	}, [setCopy, textToCopy]);

	return (
		<span onClick={onClick} role="presentation" tabIndex={-1}>
			<Popover
				placement="top"
				overlayClassName="drawer-popover"
				content={<span style={{ fontSize: '0.9rem' }}>{tooltipText}</span>}
			>
				{children}
			</Popover>
		</span>
	);
}

interface CopyClipboardHOCProps {
	entityKey: string | undefined;
	textToCopy: string;
	tooltipText?: string;
	children: ReactNode;
}

export default CopyClipboardHOC;
CopyClipboardHOC.defaultProps = {
	tooltipText: 'Copy to clipboard',
};
