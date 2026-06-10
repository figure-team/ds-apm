import { useTranslation } from 'react-i18next';
import { Collapse, Input, Typography } from 'antd';

import { useCreateAlertState } from '../context';
import AdvancedOptionItem from './AdvancedOptionItem';
import EvaluationCadence from './EvaluationCadence';

function AdvancedOptions(): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { advancedOptions, setAdvancedOptions } = useCreateAlertState();

	return (
		<div className="advanced-options-container">
			<Collapse bordered={false}>
				<Collapse.Panel header={t('v2_advanced_options')} key="1">
					<EvaluationCadence />
					<AdvancedOptionItem
						title={t('v2_alert_data_stops_title')}
						description={t('v2_alert_data_stops_desc')}
						tooltipText={t('v2_alert_data_stops_tooltip')}
						input={
							<div className="advanced-option-item-input-group">
								<Input
									placeholder={t('v2_tolerance_limit_placeholder')}
									type="number"
									style={{ width: 100 }}
									onChange={(e): void =>
										setAdvancedOptions({
											type: 'SET_SEND_NOTIFICATION_IF_DATA_IS_MISSING',
											payload: {
												toleranceLimit: Number(e.target.value),
												timeUnit: advancedOptions.sendNotificationIfDataIsMissing.timeUnit,
											},
										})
									}
									value={advancedOptions.sendNotificationIfDataIsMissing.toleranceLimit}
								/>
								<Typography.Text>{t('v2_minutes_text')}</Typography.Text>
							</div>
						}
						onToggle={(): void =>
							setAdvancedOptions({
								type: 'TOGGLE_SEND_NOTIFICATION_IF_DATA_IS_MISSING',
								payload: !advancedOptions.sendNotificationIfDataIsMissing.enabled,
							})
						}
						defaultShowInput={advancedOptions.sendNotificationIfDataIsMissing.enabled}
						data-testid="send-notification-if-data-is-missing-container"
					/>
					<AdvancedOptionItem
						title={t('v2_min_data_required_title')}
						description={t('v2_min_data_required_desc')}
						tooltipText={t('v2_min_data_required_tooltip')}
						input={
							<div className="advanced-option-item-input-group">
								<Input
									placeholder={t('v2_min_datapoints_placeholder')}
									style={{ width: 100 }}
									type="number"
									onChange={(e): void =>
										setAdvancedOptions({
											type: 'SET_ENFORCE_MINIMUM_DATAPOINTS',
											payload: {
												minimumDatapoints: Number(e.target.value),
											},
										})
									}
									value={advancedOptions.enforceMinimumDatapoints.minimumDatapoints}
								/>
								<Typography.Text>{t('v2_datapoints_text')}</Typography.Text>
							</div>
						}
						onToggle={(): void =>
							setAdvancedOptions({
								type: 'TOGGLE_ENFORCE_MINIMUM_DATAPOINTS',
								payload: !advancedOptions.enforceMinimumDatapoints.enabled,
							})
						}
						defaultShowInput={advancedOptions.enforceMinimumDatapoints.enabled}
						data-testid="enforce-minimum-datapoints-container"
					/>
					{/* TODO: Add back when the functionality is implemented */}
					{/* <AdvancedOptionItem
						title="Account for data delay"
						description="Shift the evaluation window backwards to account for data processing delays."
						tooltipText="Use when your data takes time to arrive on the platform. For example, if logs typically arrive 5 minutes late, set a 5-minute delay so the alert checks the correct time window."
						input={
							<div className="advanced-option-item-input-group">
								<Input
									placeholder="Enter delay..."
									style={{ width: 100 }}
									type="number"
									onChange={(e): void =>
										setAdvancedOptions({
											type: 'SET_DELAY_EVALUATION',
											payload: {
												delay: Number(e.target.value),
												timeUnit: advancedOptions.delayEvaluation.timeUnit,
											},
										})
									}
									value={advancedOptions.delayEvaluation.delay}
								/>
								<Select
									style={{ width: 120 }}
									options={timeOptions}
									placeholder="Select time unit"
									onChange={(value): void =>
										setAdvancedOptions({
											type: 'SET_DELAY_EVALUATION',
											payload: {
												delay: advancedOptions.delayEvaluation.delay,
												timeUnit: value as string,
											},
										})
									}
									value={advancedOptions.delayEvaluation.timeUnit}
								/>
							</div>
						}
					/> */}
				</Collapse.Panel>
			</Collapse>
		</div>
	);
}

export default AdvancedOptions;
