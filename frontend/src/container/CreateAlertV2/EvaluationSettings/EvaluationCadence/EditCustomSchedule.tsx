import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Typography } from 'antd';
import { useCreateAlertState } from 'container/CreateAlertV2/context';
import { INITIAL_ADVANCED_OPTIONS_STATE } from 'container/CreateAlertV2/context/constants';
import { IEditCustomScheduleProps } from 'container/CreateAlertV2/EvaluationSettings/types';
import { Calendar1, Edit, Trash } from 'lucide-react';

function EditCustomSchedule({
	setIsEvaluationCadenceDetailsVisible,
	setIsPreviewVisible,
}: IEditCustomScheduleProps): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { advancedOptions, setAdvancedOptions } = useCreateAlertState();

	const displayText = useMemo(() => {
		if (advancedOptions.evaluationCadence.mode === 'custom') {
			return (
				<Typography.Text>
					<Typography.Text>{t('v2_edit_custom_schedule_every')}</Typography.Text>
					<Typography.Text className="highlight">
						{advancedOptions.evaluationCadence.custom.repeatEvery
							.charAt(0)
							.toUpperCase() +
							advancedOptions.evaluationCadence.custom.repeatEvery.slice(1)}
					</Typography.Text>
					{advancedOptions.evaluationCadence.custom.repeatEvery !== 'day' && (
						<>
							<Typography.Text>{t('v2_edit_custom_schedule_on')}</Typography.Text>
							<Typography.Text className="highlight">
								{advancedOptions.evaluationCadence.custom.occurence
									.map(
										(occurence) => occurence.charAt(0).toUpperCase() + occurence.slice(1),
									)
									.join(', ')}
							</Typography.Text>
						</>
					)}
					<Typography.Text>{t('v2_edit_custom_schedule_at')}</Typography.Text>
					<Typography.Text className="highlight">
						{advancedOptions.evaluationCadence.custom.startAt}
					</Typography.Text>
				</Typography.Text>
			);
		}
		return (
			<Typography.Text>
				<Typography.Text>{t('v2_edit_custom_schedule_starting_on')}</Typography.Text>
				<Typography.Text className="highlight">
					{advancedOptions.evaluationCadence.rrule.date?.format('DD/MM/YYYY')}
				</Typography.Text>
				<Typography.Text>{t('v2_edit_custom_schedule_at')}</Typography.Text>
				<Typography.Text className="highlight">
					{advancedOptions.evaluationCadence.rrule.startAt}
				</Typography.Text>
			</Typography.Text>
		);
	}, [advancedOptions.evaluationCadence]);

	const handleEdit = (): void => {
		setIsEvaluationCadenceDetailsVisible(true);
	};

	const handlePreview = (): void => {
		setIsPreviewVisible(true);
	};

	const handleDiscard = (): void => {
		setIsEvaluationCadenceDetailsVisible(false);
		setAdvancedOptions({
			type: 'SET_EVALUATION_CADENCE',
			payload: INITIAL_ADVANCED_OPTIONS_STATE.evaluationCadence,
		});
		setAdvancedOptions({
			type: 'SET_EVALUATION_CADENCE_MODE',
			payload: 'default',
		});
	};

	return (
		<div className="edit-custom-schedule">
			{displayText}
			<div className="button-row">
				<Button.Group>
					<Button type="default" onClick={handleEdit}>
						<Edit size={12} />
						<Typography.Text>{t('v2_edit_custom_schedule_btn')}</Typography.Text>
					</Button>
					<Button type="default" onClick={handlePreview}>
						<Calendar1 size={12} />
						<Typography.Text>{t('v2_schedule_preview_btn')}</Typography.Text>
					</Button>
					<Button
						data-testid="discard-button"
						type="default"
						onClick={handleDiscard}
					>
						<Trash size={12} />
					</Button>
				</Button.Group>
			</div>
		</div>
	);
}

export default EditCustomSchedule;
