import { Trans, useTranslation } from 'react-i18next';
import { Button, Modal, Typography } from 'antd';
import { Loader, Trash2, X } from 'lucide-react';

import { DeleteRoutingPolicyProps } from './types';

function DeleteRoutingPolicy({
	handleClose,
	handleDelete,
	routingPolicy,
	isDeletingRoutingPolicy,
}: DeleteRoutingPolicyProps): JSX.Element {
	const { t } = useTranslation('alerts');
	const deleteButtonIcon = isDeletingRoutingPolicy ? (
		<Loader size={16} />
	) : (
		<Trash2 size={16} />
	);

	return (
		<Modal
			className="delete-policy-modal"
			title={<span className="title">{t('rp_delete_title')}</span>}
			open
			closable={false}
			onCancel={handleClose}
			footer={[
				<Button
					key="cancel"
					onClick={handleClose}
					className="cancel-btn"
					icon={<X size={16} />}
					disabled={isDeletingRoutingPolicy}
				>
					{t('rp_delete_cancel')}
				</Button>,
				<Button
					key="submit"
					type="primary"
					icon={deleteButtonIcon}
					onClick={handleDelete}
					className="delete-btn"
					disabled={isDeletingRoutingPolicy}
				>
					{t('rp_delete_confirm')}
				</Button>,
			]}
		>
			<Typography.Text className="delete-text">
				<Trans
					i18nKey="alerts:rp_delete_text"
					values={{ name: routingPolicy?.name }}
					components={{ 1: <strong /> }}
				/>
			</Typography.Text>
		</Modal>
	);
}

export default DeleteRoutingPolicy;
