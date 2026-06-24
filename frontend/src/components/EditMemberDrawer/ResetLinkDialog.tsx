import { useTranslation } from 'react-i18next';
import { Check, Copy } from '@signozhq/icons';
import { Button, DialogWrapper } from '@signozhq/ui';

interface ResetLinkDialogProps {
	open: boolean;
	linkType: 'invite' | 'reset' | null;
	resetLink: string | null;
	expiresAt: string | null;
	hasCopied: boolean;
	onClose: () => void;
	onCopy: () => void;
}

function ResetLinkDialog({
	open,
	linkType,
	resetLink,
	expiresAt,
	hasCopied,
	onClose,
	onCopy,
}: ResetLinkDialogProps): JSX.Element {
	const { t } = useTranslation(['organizationsettings']);

	return (
		<DialogWrapper
			open={open}
			onOpenChange={(isOpen): void => {
				if (!isOpen) {
					onClose();
				}
			}}
			title={
				linkType === 'invite'
					? t('reset_dialog_invite_title')
					: t('reset_dialog_reset_title')
			}
			showCloseButton
			width="base"
			className="reset-link-dialog"
		>
			<div className="reset-link-dialog__content">
				<p className="reset-link-dialog__description">
					{linkType === 'invite'
						? t('reset_dialog_invite_desc')
						: t('reset_dialog_reset_desc')}
				</p>
				<div className="reset-link-dialog__link-row">
					<div className="reset-link-dialog__link-text-wrap">
						<span className="reset-link-dialog__link-text">{resetLink}</span>
					</div>
					<Button
						variant="outlined"
						color="secondary"
						size="sm"
						onClick={onCopy}
						prefix={hasCopied ? <Check size={12} /> : <Copy size={12} />}
						className="reset-link-dialog__copy-btn"
					>
						{hasCopied ? t('reset_dialog_copied') : t('reset_dialog_copy')}
					</Button>
				</div>
				{expiresAt && (
					<p className="reset-link-dialog__description">
						{t('reset_dialog_expires_on', { date: expiresAt })}
					</p>
				)}
			</div>
		</DialogWrapper>
	);
}

export default ResetLinkDialog;
