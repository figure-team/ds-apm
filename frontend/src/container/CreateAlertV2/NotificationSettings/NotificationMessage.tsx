import { useTranslation } from 'react-i18next';
import { Button, Input, Popover, Tooltip, Typography } from 'antd';
import { previewNotificationTemplate } from 'api/v2/rules/previewNotificationTemplate';
import { Info } from 'lucide-react';
import { useCallback, useState } from 'react';

import { useCreateAlertState } from '../context';
import {
	getUnknownIncidentTemplateVariables,
	INCIDENT_TEMPLATE_VARIABLES,
} from './incidentTemplateVariables';

function NotificationMessage(): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { alertState, notificationSettings, setNotificationSettings } =
		useCreateAlertState();
	const [previewBody, setPreviewBody] = useState<string>('');
	const [previewMissingVars, setPreviewMissingVars] = useState<string[]>([]);
	const [previewError, setPreviewError] = useState<string>('');
	const [isPreviewLoading, setIsPreviewLoading] = useState(false);

	const unknownIncidentVariables = getUnknownIncidentTemplateVariables(
		notificationSettings.description,
	);

	const handlePreview = useCallback(async (): Promise<void> => {
		setPreviewError('');
		setIsPreviewLoading(true);

		try {
			const response = await previewNotificationTemplate({
				template: notificationSettings.description,
				labels: alertState.labels,
				annotations: alertState.annotations,
			});

			setPreviewBody(response.data.body);
			setPreviewMissingVars(response.data.missingVars || []);
		} catch {
			setPreviewBody('');
			setPreviewMissingVars([]);
			setPreviewError(t('v2_preview_error_msg'));
		} finally {
			setIsPreviewLoading(false);
		}
	}, [
		alertState.annotations,
		alertState.labels,
		notificationSettings.description,
	]);

	const handleDescriptionChange = useCallback(
		(value: string): void => {
			setPreviewBody('');
			setPreviewMissingVars([]);
			setPreviewError('');
			setNotificationSettings({
				type: 'SET_DESCRIPTION',
				payload: value,
			});
		},
		[setNotificationSettings],
	);

	const templateVariableContent = (
		<div className="template-variable-content">
			<Typography.Text strong>{t('v2_incident_template_vars_title')}</Typography.Text>
			<Typography.Text className="template-variable-content-description">
				{t('v2_incident_template_vars_desc')}
			</Typography.Text>
			{INCIDENT_TEMPLATE_VARIABLES.map((item) => (
				<div className="template-variable-content-item" key={item.variable}>
					<code>{item.variable}</code>
					<Typography.Text>{item.description}</Typography.Text>
				</div>
			))}
		</div>
	);

	return (
		<div className="notification-message-container">
			<div className="notification-message-header">
				<div className="notification-message-header-content">
					<Typography.Text className="notification-message-header-title">
						{t('v2_notification_message_title')}
						<Tooltip title={t('v2_notification_message_tooltip')}>
							<Info size={16} />
						</Tooltip>
					</Typography.Text>
					<Typography.Text className="notification-message-header-description">
						{t('v2_notification_message_desc')}
					</Typography.Text>
				</div>
				<div className="notification-message-header-actions">
					<Popover content={templateVariableContent} trigger="click">
						<Button type="text">
							<Info size={12} />
							{t('v2_variables_btn')}
						</Button>
					</Popover>
					<Button
						disabled={!notificationSettings.description.trim()}
						loading={isPreviewLoading}
						onClick={handlePreview}
						type="text"
					>
						{t('v2_preview_btn')}
					</Button>
				</div>
			</div>
			<div className="notification-message-incident-hint">
				<Typography.Text>
					{t('v2_pm_sisam_hint')}{' '}
					<code>$incident.impact_summary</code>, <code>$incident.next_action</code>,
					<code>$incident.service_name</code>, and <code>$incident.sop_id</code>.
				</Typography.Text>
			</div>
			{unknownIncidentVariables.length > 0 && (
				<div className="notification-message-warning" role="alert">
					{unknownIncidentVariables.length > 1
						? t('v2_unknown_incident_vars')
						: t('v2_unknown_incident_var')}
					:{' '}
					{unknownIncidentVariables.join(', ')}. {t('v2_unknown_vars_hint')}
				</div>
			)}
			{previewError && (
				<div className="notification-message-warning" role="alert">
					{previewError}
				</div>
			)}
			{previewBody && (
				<div className="notification-message-preview">
					<Typography.Text className="notification-message-preview-title">
						{t('v2_preview_title')}
					</Typography.Text>
					<pre>{previewBody}</pre>
					{previewMissingVars.length > 0 && (
						<div className="notification-message-warning" role="alert">
							{t('v2_preview_missing_vars')} {previewMissingVars.join(', ')}
						</div>
					)}
				</div>
			)}
			<Input.TextArea
				value={notificationSettings.description}
				onChange={(e): void => handleDescriptionChange(e.target.value)}
				placeholder={t('v2_notification_message_placeholder')}
			/>
		</div>
	);
}

export default NotificationMessage;
