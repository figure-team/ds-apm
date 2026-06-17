import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Input, Select, Tooltip, Typography } from 'antd';
import { CircleX, Trash } from 'lucide-react';
import { useAppContext } from 'providers/App/App';

import { useCreateAlertState } from '../context';
import { AlertThresholdOperator } from '../context/types';
import { normalizeOperator } from '../utils';
import { ThresholdItemProps } from './types';
import { NotificationChannelsNotFoundContent } from './utils';

function ThresholdItem({
	threshold,
	updateThreshold,
	removeThreshold,
	showRemoveButton,
	channels,
	units,
	isErrorChannels,
	refreshChannels,
	isLoadingChannels,
}: ThresholdItemProps): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { user } = useAppContext();
	const { thresholdState, notificationSettings } = useCreateAlertState();
	const [showRecoveryThreshold, setShowRecoveryThreshold] = useState(false);

	// Severity is the single source of truth: the threshold name becomes the
	// `threshold.name` label (routing key) and the auto-derived `severity` label
	// at fire time. Constrain it to the shared 4-value vocabulary, but keep any
	// legacy/custom name so editing an existing rule never silently drops it.
	const severityOptions = useMemo(() => {
		const base = ['critical', 'error', 'warning', 'info'];
		const values =
			!threshold.label || base.includes(threshold.label)
				? base
				: [...base, threshold.label];
		return values.map((value) => ({ value, label: value }));
	}, [threshold.label]);

	const yAxisUnitSelect = useMemo(() => {
		let component = (
			<Select
				placeholder={t('v2_unit_placeholder')}
				value={threshold.unit ? threshold.unit : null}
				onChange={(value): void => updateThreshold(threshold.id, 'unit', value)}
				style={{ width: 150 }}
				options={units}
				disabled={units.length === 0}
				data-testid="threshold-unit-select"
			/>
		);
		if (units.length === 0) {
			component = (
				<Tooltip trigger="hover" title={t('v2_no_compatible_units_tooltip')}>
					<Select
						placeholder={t('v2_unit_placeholder')}
						value={threshold.unit ? threshold.unit : null}
						onChange={(value): void => updateThreshold(threshold.id, 'unit', value)}
						style={{ width: 150 }}
						options={units}
						disabled={units.length === 0}
						data-testid="threshold-unit-select"
					/>
				</Tooltip>
			);
		}
		return component;
	}, [units, threshold.unit, updateThreshold, threshold.id]);

	const getOperatorSymbol = (): string => {
		switch (normalizeOperator(thresholdState.operator)) {
			case AlertThresholdOperator.IS_ABOVE:
				return '>';
			case AlertThresholdOperator.IS_BELOW:
				return '<';
			case AlertThresholdOperator.IS_EQUAL_TO:
				return '=';
			case AlertThresholdOperator.IS_NOT_EQUAL_TO:
				return '!=';
			default:
				return '';
		}
	};

	// const addRecoveryThreshold = (): void => {
	// 	setShowRecoveryThreshold(true);
	// 	updateThreshold(threshold.id, 'recoveryThresholdValue', 0);
	// };

	const removeRecoveryThreshold = (): void => {
		setShowRecoveryThreshold(false);
		updateThreshold(threshold.id, 'recoveryThresholdValue', null);
	};

	return (
		<div key={threshold.id} className="threshold-item">
			<div className="threshold-row">
				<div className="threshold-indicator">
					<div
						className="threshold-dot"
						style={{ backgroundColor: threshold.color }}
					/>
				</div>
				<div className="threshold-controls">
					<Select
						placeholder={t('v2_threshold_name_placeholder')}
						value={threshold.label || undefined}
						onChange={(value): void =>
							updateThreshold(threshold.id, 'label', value)
						}
						style={{ width: 200 }}
						options={severityOptions}
						data-testid="threshold-name-select"
					/>
					<Typography.Text className="sentence-text">{t('v2_on_value_text')}</Typography.Text>
					<Typography.Text className="sentence-text highlighted-text">
						{getOperatorSymbol()}
					</Typography.Text>
					<Input
						placeholder={t('v2_threshold_value_placeholder')}
						value={threshold.thresholdValue}
						onChange={(e): void =>
							updateThreshold(threshold.id, 'thresholdValue', e.target.value)
						}
						style={{ width: 100 }}
						type="number"
						data-testid="threshold-value-input"
					/>
					{yAxisUnitSelect}
					{!notificationSettings.routingPolicies && (
						<>
							<Typography.Text className="sentence-text">{t('v2_send_to_text')}</Typography.Text>
							<Select
								value={threshold.channels}
								onChange={(value): void =>
									updateThreshold(threshold.id, 'channels', value)
								}
								data-testid="threshold-notification-channel-select"
								style={{ width: 350 }}
								options={channels.map((channel) => ({
									value: channel.name,
									label: channel.name,
									'data-testid': `threshold-notification-channel-option-${threshold.label}`,
								}))}
								mode="multiple"
								placeholder={t('v2_notification_channels_placeholder')}
								showSearch
								maxTagCount={2}
								maxTagPlaceholder={(omittedValues): string =>
									`+${omittedValues.length} more`
								}
								maxTagTextLength={10}
								filterOption={(input, option): boolean =>
									option?.label?.toLowerCase().includes(input.toLowerCase()) || false
								}
								status={isErrorChannels ? 'error' : undefined}
								disabled={isLoadingChannels}
								notFoundContent={
									<NotificationChannelsNotFoundContent
										user={user}
										refreshChannels={refreshChannels}
									/>
								}
							/>
						</>
					)}
					{showRecoveryThreshold && (
						<>
							<Typography.Text className="sentence-text">{t('v2_recover_on_text')}</Typography.Text>
							<Input
								placeholder={t('v2_recovery_threshold_placeholder')}
								value={threshold.recoveryThresholdValue ?? ''}
								onChange={(e): void =>
									updateThreshold(threshold.id, 'recoveryThresholdValue', e.target.value)
								}
								style={{ width: 100 }}
								type="number"
								data-testid="recovery-threshold-value-input"
							/>
							<Tooltip title={t('v2_remove_recovery_threshold_tooltip')}>
								<Button
									type="default"
									icon={<Trash size={16} />}
									onClick={removeRecoveryThreshold}
									className="icon-btn"
									data-testid="remove-recovery-threshold-button"
								/>
							</Tooltip>
						</>
					)}
					<Button.Group>
						{/* TODO: Add recovery threshold back once the functionality is implemented */}
						{/* {!showRecoveryThreshold && (
							<Tooltip title="Add recovery threshold">
								<Button
									type="default"
									icon={<ChartLine size={16} />}
									className="icon-btn"
									onClick={addRecoveryThreshold}
								/>
							</Tooltip>
						)} */}
						{showRemoveButton && (
							<Tooltip title={t('v2_remove_threshold_tooltip')}>
								<Button
									type="default"
									icon={<CircleX size={16} />}
									onClick={(): void => removeThreshold(threshold.id)}
									className="icon-btn"
									data-testid="remove-threshold-button"
								/>
							</Tooltip>
						)}
					</Button.Group>
				</div>
			</div>
		</div>
	);
}

export default ThresholdItem;
