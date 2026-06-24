import { Trans, useTranslation } from 'react-i18next';
import { Trash2, X } from '@signozhq/icons';
import { Button, DialogWrapper } from '@signozhq/ui';
import { MemberRow } from 'components/MembersTable/MembersTable';

interface DeleteMemberDialogProps {
	open: boolean;
	isInvited: boolean;
	member: MemberRow | null;
	isDeleting: boolean;
	onClose: () => void;
	onConfirm: () => void;
}

function DeleteMemberDialog({
	open,
	isInvited,
	member,
	isDeleting,
	onClose,
	onConfirm,
}: DeleteMemberDialogProps): JSX.Element {
	const { t } = useTranslation(['organizationsettings']);
	const title = isInvited ? t('member_revoke_invite') : t('member_delete');

	const body = isInvited ? (
		<Trans
			t={t}
			i18nKey="member_revoke_invite_confirm"
			values={{ email: member?.email }}
			components={[<strong key="0" />]}
		/>
	) : (
		<Trans
			t={t}
			i18nKey="member_delete_confirm"
			values={{ name: member?.name || member?.email }}
			components={[<strong key="0" />]}
		/>
	);

	const footer = (
		<>
			<Button variant="solid" color="secondary" onClick={onClose}>
				<X size={12} />
				{t('cancel')}
			</Button>
			<Button
				variant="solid"
				color="destructive"
				disabled={isDeleting}
				onClick={onConfirm}
			>
				<Trash2 size={12} />
				{isDeleting ? t('member_processing') : title}
			</Button>
		</>
	);

	return (
		<DialogWrapper
			open={open}
			onOpenChange={(isOpen): void => {
				if (!isOpen) {
					onClose();
				}
			}}
			title={title}
			width="narrow"
			className="alert-dialog delete-dialog"
			showCloseButton={false}
			disableOutsideClick={false}
			footer={footer}
		>
			{body}
		</DialogWrapper>
	);
}

export default DeleteMemberDialog;
