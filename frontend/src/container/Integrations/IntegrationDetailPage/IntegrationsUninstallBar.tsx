import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from 'react-query';
import { Button, Modal, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import unInstallIntegration from 'api/Integrations/uninstallIntegration';
import { SOMETHING_WENT_WRONG } from 'constants/api';
import { useNotifications } from 'hooks/useNotifications';
import { X } from 'lucide-react';

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
