import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
	Button,
	Divider,
	Flex,
	Form,
	Input,
	Modal,
	Select,
	Typography,
} from 'antd';
import ROUTES from 'constants/routes';
import { ModalTitle } from 'container/PipelinePage/PipelineListsView/styles';
import { Check, Loader, X } from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';
import { openInNewTab } from 'utils/navigation';

import { INITIAL_ROUTING_POLICY_DETAILS_FORM_STATE } from './constants';
import {
	RoutingPolicyDetailsFormState,
	RoutingPolicyDetailsProps,
} from './types';

function RoutingPolicyDetails({
	closeModal,
	mode,
	channels,
	isErrorChannels,
	isLoadingChannels,
	routingPolicy,
	handlePolicyDetailsModalAction,
	isPolicyDetailsModalActionLoading,
	refreshChannels,
}: RoutingPolicyDetailsProps): JSX.Element {
	const { t } = useTranslation('alerts');
	const [form] = Form.useForm();
	const { user } = useAppContext();

	const initialFormState = useMemo(() => {
		if (mode === 'edit') {
			return {
				name: routingPolicy?.name || '',
				expression: routingPolicy?.expression || '',
				channels: routingPolicy?.channels || [],
				description: routingPolicy?.description || '',
			};
		}
		return INITIAL_ROUTING_POLICY_DETAILS_FORM_STATE;
	}, [routingPolicy, mode]);

	const saveButtonIcon = isPolicyDetailsModalActionLoading ? (
		<Loader size={16} />
	) : (
		<Check size={16} />
	);

	const modalTitle = mode === 'edit' ? t('rp_modal_edit') : t('rp_modal_create');

	const handleSave = (): void => {
		handlePolicyDetailsModalAction(mode, {
			name: form.getFieldValue('name'),
			expression: form.getFieldValue('expression'),
			channels: form.getFieldValue('channels'),
			description: form.getFieldValue('description'),
		});
	};

	const notificationChannelsNotFoundContent = (
		<Flex justify="space-between">
			<Flex gap={4} align="center">
				<Typography.Text>{t('rp_no_channels')}</Typography.Text>
				{user?.role === USER_ROLES.ADMIN ? (
					<Typography.Text>
						{t('rp_create_one')}
						<Button
							style={{ padding: '0 4px' }}
							type="link"
							onClick={(): void => {
								openInNewTab(ROUTES.CHANNELS_NEW);
							}}
						>
							{t('rp_here')}
						</Button>
					</Typography.Text>
				) : (
					<Typography.Text>{t('rp_ask_admin')}</Typography.Text>
				)}
			</Flex>
			<Button type="text" onClick={refreshChannels}>
				{t('rp_refresh')}
			</Button>
		</Flex>
	);

	return (
		<Modal
			title={<ModalTitle level={4}>{modalTitle}</ModalTitle>}
			centered
			open
			className="create-policy-modal"
			width={600}
			onCancel={closeModal}
			footer={null}
			maskClosable={false}
		>
			<Divider plain />
			<Form<RoutingPolicyDetailsFormState>
				form={form}
				initialValues={initialFormState}
				onFinish={handleSave}
			>
				<div className="create-policy-container">
					<div className="input-group">
						<Typography.Text>{t('rp_field_name')}</Typography.Text>
						<Form.Item
							name="name"
							rules={[
								{
									required: true,
									message: t('rp_name_required'),
								},
							]}
						>
							<Input placeholder={t('rp_name_placeholder')} />
						</Form.Item>
					</div>
					<div className="input-group">
						<Typography.Text>{t('rp_field_description')}</Typography.Text>
						<Form.Item
							name="description"
							rules={[
								{
									required: false,
								},
							]}
						>
							<Input.TextArea
								placeholder={t('rp_description_placeholder')}
								autoSize={{ minRows: 1, maxRows: 6 }}
								style={{ resize: 'none' }}
							/>
						</Form.Item>
					</div>
					<div className="input-group">
						<Typography.Text>{t('rp_field_expression')}</Typography.Text>
						<Form.Item
							name="expression"
							rules={[
								{
									required: true,
									message: t('rp_expression_required'),
								},
							]}
						>
							<Input.TextArea
								placeholder='e.g. service.name == "payment" && threshold.name == "critical"'
								autoSize={{ minRows: 1, maxRows: 6 }}
								style={{ resize: 'none' }}
							/>
						</Form.Item>
					</div>
					<div className="input-group">
						<Typography.Text>{t('rp_field_channels')}</Typography.Text>
						<Form.Item
							name="channels"
							rules={[
								{
									required: true,
									message: t('rp_channels_required'),
								},
							]}
						>
							<Select
								options={channels.map((channel) => ({
									value: channel.name,
									label: channel.name,
								}))}
								mode="multiple"
								placeholder={t('rp_channels_placeholder')}
								showSearch
								maxTagCount={3}
								maxTagPlaceholder={(omittedValues): string =>
									t('rp_more', { count: omittedValues.length })
								}
								maxTagTextLength={10}
								filterOption={(input, option): boolean =>
									option?.label?.toLowerCase().includes(input.toLowerCase()) || false
								}
								status={isErrorChannels ? 'error' : undefined}
								disabled={isLoadingChannels}
								notFoundContent={notificationChannelsNotFoundContent}
							/>
						</Form.Item>
					</div>
				</div>
				<Flex className="create-policy-footer" justify="space-between">
					<Button
						icon={<X size={16} />}
						onClick={closeModal}
						disabled={isPolicyDetailsModalActionLoading}
					>
						{t('rp_cancel')}
					</Button>
					<Button
						icon={saveButtonIcon}
						type="primary"
						htmlType="submit"
						loading={isPolicyDetailsModalActionLoading}
						disabled={isPolicyDetailsModalActionLoading}
					>
						{t('rp_save')}
					</Button>
				</Flex>
			</Form>
		</Modal>
	);
}

export default RoutingPolicyDetails;
