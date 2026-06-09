import { useTranslation } from 'react-i18next';
import { Input, Select, Typography } from 'antd';

import { useCreateAlertState } from '../context';
import {
	RE_NOTIFICATION_CONDITION_OPTIONS,
	RE_NOTIFICATION_TIME_UNIT_OPTIONS,
} from '../context/constants';
import AdvancedOptionItem from '../EvaluationSettings/AdvancedOptionItem';
import Stepper from '../Stepper';
import MultipleNotifications from './MultipleNotifications';
import NotificationMessage from './NotificationMessage';

import './styles.scss';

function NotificationSettings(): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { notificationSettings, setNotificationSettings } =
		useCreateAlertState();

	const repeatNotificationsInput = (
		<div className="repeat-notifications-input">
			<Typography.Text>{t('v2_every_text')}</Typography.Text>
			<Input
				value={notificationSettings.reNotification.value}
				placeholder={t('v2_time_interval_placeholder')}
				disabled={!notificationSettings.reNotification.enabled}
				type="number"
				onChange={(e): void => {
					setNotificationSettings({
						type: 'SET_RE_NOTIFICATION',
						payload: {
							enabled: notificationSettings.reNotification.enabled,
							value: parseInt(e.target.value, 10),
							unit: notificationSettings.reNotification.unit,
							conditions: notificationSettings.reNotification.conditions,
						},
					});
				}}
				data-testid="repeat-notifications-time-input"
			/>
			<Select
				value={notificationSettings.reNotification.unit || null}
				placeholder={t('v2_select_unit_placeholder')}
				disabled={!notificationSettings.reNotification.enabled}
				options={RE_NOTIFICATION_TIME_UNIT_OPTIONS}
				onChange={(value): void => {
					setNotificationSettings({
						type: 'SET_RE_NOTIFICATION',
						payload: {
							enabled: notificationSettings.reNotification.enabled,
							value: notificationSettings.reNotification.value,
							unit: value,
							conditions: notificationSettings.reNotification.conditions,
						},
					});
				}}
				data-testid="repeat-notifications-unit-select"
			/>
			<Typography.Text>{t('v2_while_text')}</Typography.Text>
			<Select
				mode="multiple"
				value={notificationSettings.reNotification.conditions || null}
				placeholder={t('v2_select_conditions_placeholder')}
				disabled={!notificationSettings.reNotification.enabled}
				options={RE_NOTIFICATION_CONDITION_OPTIONS}
				onChange={(value): void => {
					setNotificationSettings({
						type: 'SET_RE_NOTIFICATION',
						payload: {
							enabled: notificationSettings.reNotification.enabled,
							value: notificationSettings.reNotification.value,
							unit: notificationSettings.reNotification.unit,
							conditions: value,
						},
					});
				}}
				data-testid="repeat-notifications-conditions-select"
			/>
		</div>
	);

	return (
		<div className="notification-settings-container">
			<Stepper stepNumber={3} label={t('v2_step_notification_settings')} />
			<NotificationMessage />
			<div className="notification-settings-content">
				<MultipleNotifications />
				<AdvancedOptionItem
					title={t('v2_repeat_notifications_title')}
					description={t('v2_repeat_notifications_desc')}
					tooltipText={t('v2_repeat_notifications_tooltip')}
					input={repeatNotificationsInput}
					onToggle={(): void => {
						setNotificationSettings({
							type: 'SET_RE_NOTIFICATION',
							payload: {
								...notificationSettings.reNotification,
								enabled: !notificationSettings.reNotification.enabled,
							},
						});
					}}
					defaultShowInput={notificationSettings.reNotification.enabled}
					data-testid="repeat-notifications-container"
				/>
			</div>
		</div>
	);
}

export default NotificationSettings;
