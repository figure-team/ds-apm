import { useTranslation } from 'react-i18next';
import { Form, Input } from 'antd';
import { CloudintegrationtypesCredentialsDTO } from 'api/generated/services/sigNoz.schemas';

function RenderConnectionFields({
	isConnectionParamsLoading,
	connectionParams,
	isFormDisabled,
}: {
	isConnectionParamsLoading?: boolean;
	connectionParams?: CloudintegrationtypesCredentialsDTO | null;
	isFormDisabled?: boolean;
}): JSX.Element | null {
	const { t } = useTranslation('integrations');
	if (
		isConnectionParamsLoading ||
		(!!connectionParams?.ingestionUrl &&
			!!connectionParams?.ingestionKey &&
			!!connectionParams?.sigNozApiUrl &&
			!!connectionParams?.sigNozApiKey)
	) {
		return null;
	}

	return (
		<Form.Item name="connectionParams">
			{!connectionParams?.ingestionUrl && (
				<Form.Item
					name="ingestionUrl"
					label={t('connection_params.ingestion_url_label')}
					rules={[{ required: true, message: 'Please enter ingestion URL' }]}
				>
					<Input placeholder={t('connection_params.ingestion_url_placeholder')} disabled={isFormDisabled} />
				</Form.Item>
			)}
			{!connectionParams?.ingestionKey && (
				<Form.Item
					name="ingestionKey"
					label={t('connection_params.ingestion_key_label')}
					rules={[{ required: true, message: 'Please enter ingestion key' }]}
				>
					<Input placeholder={t('connection_params.ingestion_key_placeholder')} disabled={isFormDisabled} />
				</Form.Item>
			)}
			{!connectionParams?.sigNozApiUrl && (
				<Form.Item
					name="sigNozApiUrl"
					label={t('connection_params.api_url_label')}
					rules={[{ required: true, message: 'Please enter SigNoz API URL' }]}
				>
					<Input placeholder={t('connection_params.api_url_placeholder')} disabled={isFormDisabled} />
				</Form.Item>
			)}
			{!connectionParams?.sigNozApiKey && (
				<Form.Item
					name="sigNozApiKey"
					label={t('connection_params.api_key_label')}
					rules={[{ required: true, message: 'Please enter SigNoz API Key' }]}
				>
					<Input placeholder={t('connection_params.api_key_placeholder')} disabled={isFormDisabled} />
				</Form.Item>
			)}
		</Form.Item>
	);
}

RenderConnectionFields.defaultProps = {
	connectionParams: null,
	isFormDisabled: false,
	isConnectionParamsLoading: false,
};

export default RenderConnectionFields;
