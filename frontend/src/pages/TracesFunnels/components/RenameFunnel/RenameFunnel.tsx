import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from 'react-query';
import { Input } from 'antd';
import SignozModal from 'components/SignozModal/SignozModal';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { useRenameFunnel } from 'hooks/TracesFunnels/useFunnels';
import { useNotifications } from 'hooks/useNotifications';
import { Check, X } from 'lucide-react';

import './RenameFunnel.styles.scss';

interface RenameFunnelProps {
	isOpen: boolean;
	onClose: () => void;
	funnelId: string;
	initialName: string;
}

function RenameFunnel({
	isOpen,
	onClose,
	funnelId,
	initialName,
}: RenameFunnelProps): JSX.Element {
	const { t } = useTranslation('trace');
	const [newFunnelName, setNewFunnelName] = useState<string>(initialName);
	const renameFunnelMutation = useRenameFunnel();
	const { notifications } = useNotifications();
	const queryClient = useQueryClient();

	const handleRename = (): void => {
		renameFunnelMutation.mutate(
			{
				funnel_id: funnelId,
				funnel_name: newFunnelName,
				timestamp: new Date().getTime(),
			},
			{
				onSuccess: () => {
					notifications.success({
						message: t('funnels.rename_success'),
					});
					queryClient.invalidateQueries([REACT_QUERY_KEY.GET_FUNNELS_LIST]);
					queryClient.invalidateQueries([
						REACT_QUERY_KEY.GET_FUNNEL_DETAILS,
						funnelId,
					]);
					onClose();
				},
				onError: () => {
					notifications.error({
						message: t('funnels.rename_failed'),
					});
				},
			},
		);
	};

	const handleCancel = (): void => {
		setNewFunnelName(initialName);
		onClose();
	};

	return (
		<SignozModal
			open={isOpen}
			title={t('funnels.rename_title')}
			width={384}
			onCancel={handleCancel}
			rootClassName="funnel-modal"
			cancelText={t('funnels.cancel')}
			okText={t('funnels.rename_ok')}
			okButtonProps={{
				icon: <Check size={14} />,
				loading: renameFunnelMutation.isLoading,
				type: 'primary',
				className: 'funnel-modal__ok-btn',
				onClick: handleRename,
				disabled: newFunnelName === initialName,
			}}
			cancelButtonProps={{
				icon: <X size={14} />,
				type: 'text',
				className: 'funnel-modal__cancel-btn',
				onClick: handleCancel,
			}}
			getContainer={document.getElementById('root') || undefined}
			destroyOnClose
		>
			<div className="funnel-modal-content">
				<span className="funnel-modal-content__label">{t('funnels.rename_label')}</span>
				<Input
					className="funnel-modal-content__input"
					value={newFunnelName}
					onChange={(e): void => setNewFunnelName(e.target.value)}
					autoFocus
				/>
			</div>
		</SignozModal>
	);
}

export default RenameFunnel;
