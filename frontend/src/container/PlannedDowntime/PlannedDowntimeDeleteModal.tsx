import { SetStateAction } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Modal, Typography } from 'antd';
import { Trash2, X } from 'lucide-react';

import './PlannedDowntime.styles.scss';

interface PlannedDowntimeDeleteModalProps {
	isDeleteModalOpen: boolean;
	setIsDeleteModalOpen: (value: SetStateAction<boolean>) => void;
	onDeleteHandler: () => void;
	isDeleteLoading: boolean;
	downtimeSchedule: string;
}

export function PlannedDowntimeDeleteModal(
	props: PlannedDowntimeDeleteModalProps,
): JSX.Element {
	const {
		isDeleteModalOpen,
		setIsDeleteModalOpen,
		isDeleteLoading,
		onDeleteHandler,
		downtimeSchedule,
	} = props;
	const { t } = useTranslation('alerts');
	const hideDeleteScheduleModal = (): void => {
		setIsDeleteModalOpen(false);
	};
	return (
		<Modal
			className="delete-schedule-modal"
			title={<span className="title">{t('pd_delete_title')}</span>}
			open={isDeleteModalOpen}
			closable={false}
			onCancel={hideDeleteScheduleModal}
			footer={[
				<Button
					key="cancel"
					onClick={hideDeleteScheduleModal}
					className="cancel-btn"
					icon={<X size={16} />}
				>
					{t('pd_delete_cancel')}
				</Button>,
				<Button
					key="submit"
					icon={<Trash2 size={16} />}
					onClick={onDeleteHandler}
					className="delete-btn"
					disabled={isDeleteLoading}
				>
					{t('pd_delete_confirm_btn')}
				</Button>,
			]}
		>
			<Typography.Text className="delete-text">
				{t('pd_delete_text', { name: downtimeSchedule })}
			</Typography.Text>
		</Modal>
	);
}
