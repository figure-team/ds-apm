import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Select, Tooltip, Typography } from 'antd';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { Info } from 'lucide-react';

import { ALL_SELECTED_VALUE } from '../constants';
import { useCreateAlertState } from '../context';

function MultipleNotifications(): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { notificationSettings, setNotificationSettings } =
		useCreateAlertState();
	const { currentQuery } = useQueryBuilder();

	const isAllOptionSelected = useMemo(
		() =>
			notificationSettings.multipleNotifications?.includes(ALL_SELECTED_VALUE),
		[notificationSettings.multipleNotifications],
	);

	const spaceAggregationOptions = useMemo(() => {
		const allGroupBys = currentQuery.builder.queryData?.reduce<string[]>(
			(acc, query) => {
				const groupByKeys = query.groupBy?.map((groupBy) => groupBy.key) || [];
				return [...acc, ...groupByKeys];
			},
			[],
		);
		const uniqueGroupBys = [...new Set(allGroupBys)];
		const options = uniqueGroupBys.map((key) => ({
			label: key,
			value: key,
			disabled: isAllOptionSelected,
			'data-testid': 'multiple-notifications-select-option',
		}));
		if (options.length > 0) {
			return [
				{
					label: t('v2_all_option'),
					value: ALL_SELECTED_VALUE,
					'data-testid': 'multiple-notifications-select-option',
				},
				...options,
			];
		}
		return options;
	}, [currentQuery.builder.queryData, isAllOptionSelected]);

	const isMultipleNotificationsEnabled = spaceAggregationOptions.length > 0;

	const onSelectChange = useCallback(
		(newSelectedOptions: string[]): void => {
			const currentSelectedOptions = notificationSettings.multipleNotifications;
			const allOptionLastSelected =
				!currentSelectedOptions?.includes(ALL_SELECTED_VALUE) &&
				newSelectedOptions.includes(ALL_SELECTED_VALUE);
			if (allOptionLastSelected) {
				setNotificationSettings({
					type: 'SET_MULTIPLE_NOTIFICATIONS',
					payload: [ALL_SELECTED_VALUE],
				});
			} else {
				setNotificationSettings({
					type: 'SET_MULTIPLE_NOTIFICATIONS',
					payload: newSelectedOptions,
				});
			}
		},
		[setNotificationSettings, notificationSettings.multipleNotifications],
	);

	const groupByDescription = useMemo(() => {
		if (isAllOptionSelected) {
			return t('v2_all_grouping_disabled');
		}
		if (notificationSettings.multipleNotifications?.length) {
			return t('v2_alerts_grouped', {
				fields: notificationSettings.multipleNotifications?.join(', '),
			});
		}
		return t('v2_empty_grouping');
	}, [isAllOptionSelected, notificationSettings.multipleNotifications, t]);

	const multipleNotificationsInput = useMemo(() => {
		const placeholder = isMultipleNotificationsEnabled
			? t('v2_select_fields_placeholder')
			: t('v2_no_grouping_fields_placeholder');
		let input = (
			<div>
				<Select
					options={spaceAggregationOptions}
					onChange={onSelectChange}
					value={notificationSettings.multipleNotifications}
					mode="multiple"
					placeholder={placeholder}
					disabled={!isMultipleNotificationsEnabled}
					aria-disabled={!isMultipleNotificationsEnabled}
					maxTagCount={3}
					data-testid="multiple-notifications-select"
				/>
				{isMultipleNotificationsEnabled && (
					<Typography.Paragraph className="multiple-notifications-select-description">
						{groupByDescription}
					</Typography.Paragraph>
				)}
			</div>
		);
		if (!isMultipleNotificationsEnabled) {
			input = (
				<Tooltip title={t('v2_enable_grouping_tooltip')}>
					{input}
				</Tooltip>
			);
		}
		return input;
	}, [
		groupByDescription,
		isMultipleNotificationsEnabled,
		notificationSettings.multipleNotifications,
		onSelectChange,
		spaceAggregationOptions,
	]);

	return (
		<div className="multiple-notifications-container">
			<div className="multiple-notifications-header">
				<Typography.Text className="multiple-notifications-header-title">
					{t('v2_group_alerts_by_title')}{' '}
					<Tooltip title={t('v2_group_alerts_by_tooltip')}>
						<Info size={16} />
					</Tooltip>
				</Typography.Text>
				<Typography.Text className="multiple-notifications-header-description">
					{t('v2_group_alerts_by_desc')}
				</Typography.Text>
			</div>
			{multipleNotificationsInput}
		</div>
	);
}

export default MultipleNotifications;
