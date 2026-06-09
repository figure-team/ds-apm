import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from 'react-query';
import { Input } from 'antd';
import SignozModal from 'components/SignozModal/SignozModal';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { useSaveFunnelDescription } from 'hooks/TracesFunnels/useFunnels';
import { useNotifications } from 'hooks/useNotifications';
import { Check, X } from 'lucide-react';

import './AddFunnelDescriptionModal.styles.scss';

interface AddFunnelDescriptionProps {
	isOpen: boolean;
	onClose: () => void;
	funnelId: string;
	funnelDescription: string;
}

function AddFunnelDescriptionModal({
	isOpen,
	onClose,
	funnelId,
	funnelDescription,
}: AddFunnelDescriptionProps): JSX.Element {
	const { t } = useTranslation('trace');
	const [description, setDescription] = useState<string>(funnelDescription);
	const { notifications } = useNotifications();
	const queryClient = useQueryClient();

	const { mutate: saveFunnelDescription, isLoading } =
		useSaveFunnelDescription();

	const handleCancel = (): void => {
		setDescription('');
		onClose();
	};

	const handleSave = (): void => {
		saveFunnelDescription(
			{
				funnel_id: funnelId,
				description,
				timestamp: Date.now(),
			},
			{
				onSuccess: () => {
					queryClient.invalidateQueries([
						REACT_QUERY_KEY.GET_FUNNEL_DETAILS,
						funnelId,
					]);
					notifications.success({
						message: t('funnels.success'),
						description: t('funnels.desc_saved'),
					});
					handleCancel();
				},
				onError: (error) => {
					notifications.error({
						message: t('funnels.desc_save_failed'),
						description: error.message,
					});
				},
			},
		);
	};

	return (
		<SignozModal
			open={isOpen}
			title={t('funnels.add_desc_title')}
			width={384}
			onCancel={handleCancel}
			rootClassName="funnel-step-modal funnel-modal signoz-modal"
			cancelText={t('funnels.cancel')}
			okText={t('funnels.save_changes')}
			okButtonProps={{
				icon: <Check size={14} />,
				type: 'primary',
				className: 'funnel-step-modal__ok-btn',
				onClick: handleSave,
				loading: isLoading,
			}}
			cancelButtonProps={{
				icon: <X size={14} />,
				type: 'text',
				className: 'funnel-step-modal__cancel-btn',
				onClick: handleCancel,
				disabled: isLoading,
			}}
			destroyOnClose
		>
			<div className="funnel-step-modal-content">
				<div className="funnel-step-modal-content__field">
					<span className="funnel-step-modal-content__label">{t('funnels.description')}</span>
					<Input.TextArea
						className="funnel-step-modal-content__input"
						placeholder="(Optional) Eg. checkout dropoff funnel"
						value={description}
						onChange={(e): void => setDescription(e.target.value)}
						autoSize={{ minRows: 3, maxRows: 5 }}
						disabled={isLoading}
					/>
				</div>
			</div>
		</SignozModal>
	);
}

export default AddFunnelDescriptionModal;
