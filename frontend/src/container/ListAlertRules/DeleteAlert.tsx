import { Dispatch, SetStateAction, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import type { NotificationInstance } from 'antd/es/notification/interface';
import { convertToApiError } from 'api/ErrorResponseHandlerForGeneratedAPIs';
import { deleteRuleByID } from 'api/generated/services/rules';
import type {
	RenderErrorResponseDTO,
	RuletypesRuleDTO,
} from 'api/generated/services/sigNoz.schemas';
import { AxiosError } from 'axios';
import { State } from 'hooks/useFetch';
import { useErrorModal } from 'providers/ErrorModalProvider';
import { PayloadProps as DeleteAlertPayloadProps } from 'types/api/alerts/delete';
import APIError from 'types/api/error';

import { ColumnButton } from './styles';

function DeleteAlert({
	id,
	alertName,
	setData,
	notifications,
}: DeleteAlertProps): JSX.Element {
	const [deleteAlertState, setDeleteAlertState] = useState<
		State<DeleteAlertPayloadProps>
	>({
		error: false,
		errorMessage: '',
		loading: false,
		success: false,
		payload: undefined,
	});

	const [modal, contextHolder] = Modal.useModal();
	const { t } = useTranslation(['alerts', 'common']);
	const { showErrorModal } = useErrorModal();

	const onDeleteHandler = async (id: string): Promise<void> => {
		try {
			await deleteRuleByID({ id });

			setData((state) => state.filter((alert) => alert.id !== id));

			setDeleteAlertState((state) => ({
				...state,
				loading: false,
			}));
			notifications.success({
				message: t('common:success'),
			});
		} catch (error) {
			setDeleteAlertState((state) => ({
				...state,
				loading: false,
				error: true,
			}));

			showErrorModal(
				convertToApiError(error as AxiosError<RenderErrorResponseDTO>) as APIError,
			);
		}
	};

	const onClickHandler = (): void => {
		modal.confirm({
			title: t('list_delete_title'),
			content: t('list_delete_confirm', { name: alertName }),
			icon: (
				<ExclamationCircleOutlined style={{ color: 'var(--danger-background)' }} />
			),
			okText: t('list_delete'),
			okButtonProps: { danger: true },
			cancelText: t('list_delete_cancel'),
			centered: true,
			onOk: () => {
				setDeleteAlertState((state) => ({
					...state,
					loading: true,
				}));
				return onDeleteHandler(id);
			},
		});
	};

	return (
		<>
			<ColumnButton
				disabled={deleteAlertState.loading || false}
				loading={deleteAlertState.loading || false}
				onClick={onClickHandler}
				type="link"
			>
				{t('list_delete')}
			</ColumnButton>
			{contextHolder}
		</>
	);
}

interface DeleteAlertProps {
	id: string;
	alertName: string;
	setData: Dispatch<SetStateAction<RuletypesRuleDTO[]>>;
	notifications: NotificationInstance;
}

export default DeleteAlert;
