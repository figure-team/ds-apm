import { useCallback, useMemo, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { PlusOutlined } from '@ant-design/icons';
import { Trash2, X } from '@signozhq/icons';
import { Button, toast } from '@signozhq/ui';
import { Modal, Table, TableColumnsType as ColumnsType } from 'antd';
import { ErrorResponseHandlerForGeneratedAPIs } from 'api/ErrorResponseHandlerForGeneratedAPIs';
import {
	useDeleteAuthDomain,
	useListAuthDomains,
} from 'api/generated/services/authdomains';
import {
	AuthtypesGettableAuthDomainDTO,
	RenderErrorResponseDTO,
} from 'api/generated/services/sigNoz.schemas';
import { AxiosError } from 'axios';
import ErrorContent from 'components/ErrorModal/components/ErrorContent';
import CopyToClipboard from 'periscope/components/CopyToClipboard';
import { useErrorModal } from 'providers/ErrorModalProvider';
import APIError from 'types/api/error';
import { getAbsoluteUrl } from 'utils/basePath';

import CreateEdit from './CreateEdit/CreateEdit';
import SSOEnforcementToggle from './SSOEnforcementToggle';

import './AuthDomain.styles.scss';
import '../../IngestionSettings/IngestionSettings.styles.scss';

export const SSOType = new Map<string, string>([
	['google_auth', 'Google Auth'],
	['saml', 'SAML'],
	['email_password', 'Email Password'],
	['oidc', 'OIDC'],
]);

function AuthDomain(): JSX.Element {
	const { t } = useTranslation(['organizationsettings', 'common']);
	const [record, setRecord] = useState<AuthtypesGettableAuthDomainDTO>();
	const [addDomain, setAddDomain] = useState<boolean>(false);
	const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
	const [activeDomain, setActiveDomain] =
		useState<AuthtypesGettableAuthDomainDTO | null>(null);

	const { showErrorModal } = useErrorModal();

	const {
		data: authDomainListResponse,
		isLoading: isLoadingAuthDomainListResponse,
		isFetching: isFetchingAuthDomainListResponse,
		error: errorFetchingAuthDomainListResponse,
		refetch: refetchAuthDomainListResponse,
	} = useListAuthDomains();

	const { mutate: deleteAuthDomain, isLoading } =
		useDeleteAuthDomain<AxiosError<RenderErrorResponseDTO>>();

	const showDeleteModal = useCallback(
		(domain: AuthtypesGettableAuthDomainDTO): void => {
			setActiveDomain(domain);
			setIsDeleteModalOpen(true);
		},
		[],
	);

	const hideDeleteModal = useCallback((): void => {
		setIsDeleteModalOpen(false);
		setActiveDomain(null);
	}, []);

	const handleDeleteDomain = useCallback((): void => {
		if (!activeDomain?.id) {
			return;
		}

		deleteAuthDomain(
			{ pathParams: { id: activeDomain.id } },
			{
				onSuccess: () => {
					toast.success(t('domain_deleted_success'));
					refetchAuthDomainListResponse();
					hideDeleteModal();
				},
				onError: (error) => {
					try {
						ErrorResponseHandlerForGeneratedAPIs(error);
					} catch (apiError) {
						showErrorModal(apiError as APIError);
					}
				},
			},
		);
	}, [
		activeDomain,
		deleteAuthDomain,
		hideDeleteModal,

		refetchAuthDomainListResponse,
		showErrorModal,
		t,
	]);

	const formattedError = useMemo(() => {
		if (!errorFetchingAuthDomainListResponse) {
			return null;
		}

		let errorResult: APIError | null = null;
		try {
			ErrorResponseHandlerForGeneratedAPIs(errorFetchingAuthDomainListResponse);
		} catch (error) {
			errorResult = error as APIError;
		}
		return errorResult;
	}, [errorFetchingAuthDomainListResponse]);

	const columns: ColumnsType<AuthtypesGettableAuthDomainDTO> = useMemo(
		() => [
			{
				title: t('domain_column_domain'),
				dataIndex: 'name',
				key: 'name',
				width: 100,
				render: (val): JSX.Element => <span>{val}</span>,
			},
			{
				title: t('domain_column_enforce_sso'),
				dataIndex: 'ssoEnabled',
				key: 'ssoEnabled',
				width: 80,
				render: (
					value: boolean,
					record: AuthtypesGettableAuthDomainDTO,
				): JSX.Element => (
					<SSOEnforcementToggle isDefaultChecked={value} record={record} />
				),
			},
			{
				title: t('domain_column_idp_sso_url'),
				dataIndex: 'relayState',
				key: 'relayState',
				width: 80,
				render: (_, record: AuthtypesGettableAuthDomainDTO): JSX.Element => {
					const relayPath = record.authNProviderInfo?.relayStatePath;
					if (!relayPath) {
						return <span className="auth-domain-list-na">{t('domain_na')}</span>;
					}

					const href = getAbsoluteUrl(`/${relayPath}`);
					return <CopyToClipboard textToCopy={href} />;
				},
			},
			{
				title: t('domain_column_action'),
				dataIndex: 'action',
				key: 'action',
				width: 100,
				render: (_, record: AuthtypesGettableAuthDomainDTO): JSX.Element => (
					<section className="auth-domain-list-column-action">
						<Button
							className="auth-domain-list-action-link"
							onClick={(): void => setRecord(record)}
							variant="link"
						>
							{t('domain_configure_provider', {
								provider: SSOType.get(record.ssoType || ''),
							})}
						</Button>
						<Button
							className="auth-domain-list-action-link delete"
							onClick={(): void => showDeleteModal(record)}
							variant="link"
						>
							{t('domain_delete')}
						</Button>
					</section>
				),
			},
		],
		[showDeleteModal, t],
	);

	return (
		<div className="auth-domain">
			<section className="auth-domain-header">
				<h3 className="auth-domain-title">{t('authenticated_domains')}</h3>
				<Button
					prefix={<PlusOutlined />}
					onClick={(): void => {
						setAddDomain(true);
					}}
					variant="solid"
					size="sm"
					color="primary"
				>
					{t('add_domain')}
				</Button>
			</section>
			{formattedError && <ErrorContent error={formattedError} />}
			{!errorFetchingAuthDomainListResponse && (
				<Table
					columns={columns}
					dataSource={authDomainListResponse?.data}
					onRow={undefined}
					loading={
						isLoadingAuthDomainListResponse || isFetchingAuthDomainListResponse
					}
					className="auth-domain-list"
					rowKey="id"
				/>
			)}
			{(addDomain || record) && (
				<CreateEdit
					isCreate={!record}
					record={record}
					onClose={(): void => {
						setAddDomain(false);
						setRecord(undefined);
						refetchAuthDomainListResponse();
					}}
				/>
			)}

			<Modal
				className="delete-ingestion-key-modal"
				title={<span className="title">{t('delete_domain')}</span>}
				open={isDeleteModalOpen}
				closable
				onCancel={hideDeleteModal}
				destroyOnClose
				footer={[
					<Button
						key="cancel"
						onClick={hideDeleteModal}
						className="cancel-btn"
						prefix={<X size={16} />}
					>
						{t('common:cancel')}
					</Button>,
					<Button
						key="submit"
						prefix={<Trash2 size={16} />}
						onClick={handleDeleteDomain}
						className="delete-btn"
						loading={isLoading}
					>
						{t('delete_domain')}
					</Button>,
				]}
			>
				<p className="delete-text">
					<Trans
						t={t}
						i18nKey="domain_delete_confirm"
						values={{ name: activeDomain?.name }}
						components={[<strong key="0" />]}
					/>
				</p>
			</Modal>
		</div>
	);
}

export default AuthDomain;
