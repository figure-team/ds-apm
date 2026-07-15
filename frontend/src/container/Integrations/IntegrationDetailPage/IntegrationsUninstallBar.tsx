import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from 'react-query';
import { Button, Modal, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import unInstallIntegration from 'api/Integrations/uninstallIntegration';
import { SOMETHING_WENT_WRONG } from 'constants/api';
import useComponentPermission from 'hooks/useComponentPermission';
import { useNotifications } from 'hooks/useNotifications';
import { X } from 'lucide-react';
import { useAppContext } from 'providers/App/App';

import { INTEGRATION_TELEMETRY_EVENTS } from '../constants';
import { ConnectionStates } from './TestConnection';

import './IntegrationDetailPage.styles.scss';

const DEFAULT_REMOVE_INTEGRATION_TITLE = 'Remove from SigNoz';

interface IntergrationsUninstallBarProps {
	integrationTitle: string;
	integrationId: string;
	onUnInstallSuccess: () => void;
	connectionStatus: ConnectionStates;
	removeIntegrationTitle?: string;
}
function IntergrationsUninstallBar(
	props: IntergrationsUninstallBarProps,
): JSX.Element {
	const {
		integrationTitle,
		integrationId,
		onUnInstallSuccess,
		connectionStatus,
		removeIntegrationTitle = DEFAULT_REMOVE_INTEGRATION_TITLE,
	} = props;
	const { notifications } = useNotifications();
	const { user } = useAppContext();
	const [uninstallPermission] = useComponentPermission(
		['uninstall_integration'],
		user.role,
	);
	const { t } = useTranslation('integrations');
	const [isModalOpen, setIsModalOpen] = useState(false);

	const removeIntegrationLabel =
		removeIntegrationTitle === DEFAULT_REMOVE_INTEGRATION_TITLE
			? t('uninstall.remove_from_signoz')
			: removeIntegrationTitle;

	const { mutate: uninstallIntegration, isLoading: isUninstallLoading } =
		useMutation(unInstallIntegration, {
			onSuccess: () => {
				onUnInstallSuccess?.();
				setIsModalOpen(false);
			},
			onError: () => {
				notifications.error({
					message: SOMETHING_WENT_WRONG,
				});
			},
		});

	const showModal = (): void => {
		setIsModalOpen(true);
	};

	const handleOk = (): void => {
		logEvent(
			INTEGRATION_TELEMETRY_EVENTS.INTEGRATIONS_DETAIL_REMOVE_INTEGRATION,
			{
				integration: integrationId,
				integrationStatus: connectionStatus,
			},
		);
		uninstallIntegration({
			integration_id: integrationId,
		});
	};

	const handleCancel = (): void => {
		setIsModalOpen(false);
	};

	// 뷰어 숨김: 백엔드 /integrations/uninstall이 EditAccess로 막혀 있고,
	// 여기 숨김은 UX 정합용이다.
	if (!uninstallPermission) {
		return <></>;
	}

	return (
		<div className="uninstall-integration-bar">
			<div className="unintall-integration-bar-text">
				<Typography.Text className="heading">
					{t('uninstall.remove_integration')}
				</Typography.Text>
				<Typography.Text className="subtitle">
					{t('uninstall.subtitle', { title: integrationTitle })}
				</Typography.Text>
			</div>
			<Button
				className="uninstall-integration-btn"
				icon={<X size={14} />}
				onClick={(): void => showModal()}
			>
				{removeIntegrationLabel}
			</Button>
			<Modal
				className="remove-integration-modal"
				open={isModalOpen}
				title={t('uninstall.modal_title')}
				onOk={handleOk}
				onCancel={handleCancel}
				okText={t('uninstall.remove_integration_btn')}
				okButtonProps={{
					danger: true,
					disabled: isUninstallLoading,
				}}
			>
				<Typography.Text className="remove-integration-text">
					{t('uninstall.remove_text', { title: integrationTitle })}
				</Typography.Text>
			</Modal>
		</div>
	);
}

IntergrationsUninstallBar.defaultProps = {
	removeIntegrationTitle: DEFAULT_REMOVE_INTEGRATION_TITLE,
};

export default IntergrationsUninstallBar;
