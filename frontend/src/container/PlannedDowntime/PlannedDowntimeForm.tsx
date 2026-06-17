import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { CheckOutlined } from '@ant-design/icons';
import {
	Button,
	DatePicker,
	Form,
	FormInstance,
	Input,
	Modal,
	Select,
	SelectProps,
	Spin,
	Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import { convertToApiError } from 'api/ErrorResponseHandlerForGeneratedAPIs';
import {
	createDowntimeSchedule,
	updateDowntimeScheduleByID,
} from 'api/generated/services/downtimeschedules';
import type {
	RuletypesPlannedMaintenanceDTO,
	RuletypesPostablePlannedMaintenanceDTO,
	RuletypesRecurrenceDTO,
} from 'api/generated/services/sigNoz.schemas';
import { RenderErrorResponseDTO } from 'api/generated/services/sigNoz.schemas';
import { AxiosError } from 'axios';
import { DATE_TIME_FORMATS } from 'constants/dateTimeFormats';
import {
	ModalButtonWrapper,
	ModalTitle,
} from 'container/PipelinePage/PipelineListsView/styles';
import dayjs from 'dayjs';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';
import { useNotifications } from 'hooks/useNotifications';
import { defaultTo, isEmpty } from 'lodash-es';
import { useErrorModal } from 'providers/ErrorModalProvider';
import APIError from 'types/api/error';
import { ALL_TIME_ZONES } from 'utils/timeZoneUtil';

import 'dayjs/locale/en';

import { AlertRuleTags } from './PlannedDowntimeList';
import {
	getAlertOptionsFromIds,
	getDurationInfo,
	getEndTime,
	handleTimeConversion,
	isScheduleRecurring,
	recurrenceOptions,
	recurrenceOptionWithSubmenu,
	recurrenceWeeklyOptions,
	translateRecurrenceType,
	translateWeekday,
} from './PlannedDowntimeutils';

import './PlannedDowntime.styles.scss';

dayjs.locale('en');
dayjs.extend(utc);
dayjs.extend(timezone);

const TIME_FORMAT = DATE_TIME_FORMATS.TIME;
const DATE_FORMAT = DATE_TIME_FORMATS.ORDINAL_DATE;
const ORDINAL_FORMAT = DATE_TIME_FORMATS.ORDINAL_ONLY;

interface PlannedDowntimeFormData {
	name: string;
	startTime: dayjs.Dayjs | string;
	endTime: dayjs.Dayjs | string;
	recurrence?: RuletypesRecurrenceDTO | null;
	alertRules: DefaultOptionType[];
	recurrenceSelect?: RuletypesRecurrenceDTO;
	timezone?: string;
}

const customFormat = DATE_TIME_FORMATS.ORDINAL_DATETIME;

interface PlannedDowntimeFormProps {
	initialValues: Partial<
		RuletypesPlannedMaintenanceDTO & {
			editMode: boolean;
		}
	>;
	alertOptions: DefaultOptionType[];
	isError: boolean;
	isLoading: boolean;
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	refetchAllSchedules: () => void;
	isEditMode: boolean;
	form: FormInstance<any>;
}

export function PlannedDowntimeForm(
	props: PlannedDowntimeFormProps,
): JSX.Element {
	const {
		initialValues,
		alertOptions,
		isError,
		isLoading,
		isOpen,
		setIsOpen,
		refetchAllSchedules,
		isEditMode,
		form,
	} = props;
	const { t } = useTranslation('alerts');

	const [selectedTags, setSelectedTags] = React.useState<
		DefaultOptionType | DefaultOptionType[]
	>([]);
	const alertRuleFormName = 'alertRules';
	const [saveLoading, setSaveLoading] = useState(false);
	const [durationUnit, setDurationUnit] = useState<string>(
		getDurationInfo(initialValues.schedule?.recurrence?.duration as string)
			?.unit || 'm',
	);

	const [formData, setFormData] = useState<Partial<PlannedDowntimeFormData>>({
		timezone: initialValues.schedule?.timezone,
	});

	const [recurrenceType, setRecurrenceType] = useState<string | null>(
		(initialValues.schedule?.recurrence?.repeatType as string) ||
			recurrenceOptions.doesNotRepeat.value,
	);

	const timezoneInitialValue = !isEmpty(initialValues.schedule?.timezone)
		? (initialValues.schedule?.timezone as string)
		: undefined;

	const { notifications } = useNotifications();
	const { showErrorModal } = useErrorModal();

	const datePickerFooter = (mode: any): any =>
		mode === 'time' ? (
			<span style={{ color: 'gray' }}>{t('pd_please_select_time')}</span>
		) : null;

	const saveHanlder = useCallback(
		async (values: PlannedDowntimeFormData) => {
			const shouldKeepLocalTime = !isEditMode;
			const data: RuletypesPostablePlannedMaintenanceDTO = {
				alertIds: values.alertRules
					.map((alert) => alert.value)
					.filter((alert) => alert !== undefined) as string[],
				name: values.name,
				schedule: {
					startTime: new Date(
						handleTimeConversion(
							values.startTime,
							timezoneInitialValue,
							values.timezone,
							shouldKeepLocalTime,
						),
					),
					timezone: values.timezone as string,
					endTime: values.endTime
						? new Date(
								handleTimeConversion(
									values.endTime,
									timezoneInitialValue,
									values.timezone,
									shouldKeepLocalTime,
								),
							)
						: undefined,
					recurrence: values.recurrence as RuletypesRecurrenceDTO,
				},
			};

			setSaveLoading(true);
			try {
				if (isEditMode && initialValues.id) {
					await updateDowntimeScheduleByID({ id: initialValues.id }, data);
				} else {
					await createDowntimeSchedule(data);
				}
				setIsOpen(false);
				notifications.success({
					message: t('pd_toast_success'),
					description: isEditMode
						? t('pd_schedule_updated')
						: t('pd_schedule_created'),
				});
				refetchAllSchedules();
			} catch (e: unknown) {
				showErrorModal(
					convertToApiError(e as AxiosError<RenderErrorResponseDTO>) as APIError,
				);
			}
			setSaveLoading(false);
		},
		[
			initialValues.id,
			isEditMode,
			notifications,
			refetchAllSchedules,
			setIsOpen,
			timezoneInitialValue,
			showErrorModal,
			t,
		],
	);
	const onFinish = async (values: PlannedDowntimeFormData): Promise<void> => {
		const recurrenceData =
			values?.recurrence?.repeatType === recurrenceOptions.doesNotRepeat.value
				? undefined
				: {
						duration: values.recurrence?.duration
							? `${values.recurrence?.duration}${durationUnit}`
							: undefined,
						endTime: !isEmpty(values.endTime)
							? handleTimeConversion(
									values.endTime,
									timezoneInitialValue,
									values.timezone,
									!isEditMode,
								)
							: undefined,
						startTime: handleTimeConversion(
							values.startTime,
							timezoneInitialValue,
							values.timezone,
							!isEditMode,
						),
						repeatOn: !values.recurrence?.repeatOn?.length
							? undefined
							: values.recurrence?.repeatOn,
						repeatType: values.recurrence?.repeatType,
					};

		const payloadValues = {
			...values,
			recurrence: recurrenceData as RuletypesRecurrenceDTO | undefined,
		};
		await saveHanlder(payloadValues);
	};

	const formValidationRules = [
		{
			required: true,
		},
	];

	const handleOk = async (): Promise<void> => {
		await form.validateFields().catch(() => {
			// antd renders inline field-level errors; nothing more to do here.
		});
	};

	const handleCancel = (): void => {
		setIsOpen(false);
	};

	const handleChange = (
		_value: string,
		options: DefaultOptionType | DefaultOptionType[],
	): void => {
		form.setFieldValue(alertRuleFormName, options);
		setSelectedTags(options);
	};

	const noTagRenderer: SelectProps['tagRender'] = () => <></>;

	const handleClose = (removedTag: DefaultOptionType['value']): void => {
		if (!removedTag) {
			return;
		}
		const newTags = selectedTags.filter(
			(tag: DefaultOptionType) => tag.value !== removedTag,
		);
		form.setFieldValue(alertRuleFormName, newTags);
		setSelectedTags(newTags);
	};

	const formatedInitialValues = useMemo(() => {
		const formData: PlannedDowntimeFormData = {
			name: defaultTo(initialValues.name, ''),
			alertRules: getAlertOptionsFromIds(
				initialValues.alertIds || [],
				alertOptions,
			),
			endTime: getEndTime(initialValues) ? dayjs(getEndTime(initialValues)) : '',
			startTime: initialValues.schedule?.startTime
				? dayjs(initialValues.schedule?.startTime)
				: '',
			recurrence: {
				...initialValues.schedule?.recurrence,
				repeatType: (!isScheduleRecurring(initialValues?.schedule)
					? recurrenceOptions.doesNotRepeat.value
					: initialValues.schedule?.recurrence
							?.repeatType) as RuletypesRecurrenceDTO['repeatType'],
				duration: String(
					getDurationInfo(initialValues.schedule?.recurrence?.duration as string)
						?.value ?? '',
				),
			} as RuletypesRecurrenceDTO,
			timezone: initialValues.schedule?.timezone as string,
		};
		return formData;
	}, [initialValues, alertOptions]);

	useEffect(() => {
		setSelectedTags(formatedInitialValues.alertRules);
		form.setFieldsValue({ ...formatedInitialValues });
	}, [form, formatedInitialValues, initialValues]);

	const timeZoneItems: DefaultOptionType[] = ALL_TIME_ZONES.map(
		(timezone: string) => ({
			label: timezone,
			value: timezone,
			key: timezone,
		}),
	);

	const getTimezoneFormattedTime = (
		time: string | dayjs.Dayjs,
		timeZone?: string,
		isEditMode?: boolean,
		format?: string,
	): string => {
		if (!time) {
			return '';
		}
		if (!timeZone) {
			return dayjs(time).format(format);
		}
		return dayjs(time).tz(timeZone, isEditMode).format(format);
	};

	const startTimeText = useMemo((): string => {
		let startTime = formData?.startTime;
		if (recurrenceType !== recurrenceOptions.doesNotRepeat.value) {
			startTime =
				(formData?.recurrence?.startTime
					? dayjs(formData.recurrence.startTime).toISOString()
					: '') ||
				formData?.startTime ||
				'';
		}

		if (!startTime) {
			return '';
		}

		if (formData.timezone) {
			startTime = handleTimeConversion(
				startTime,
				timezoneInitialValue,
				formData?.timezone,
				!isEditMode,
			);
		}
		const daysOfWeek = formData?.recurrence?.repeatOn;

		const formattedStartTime = getTimezoneFormattedTime(
			startTime,
			formData.timezone,
			!isEditMode,
			TIME_FORMAT,
		);

		const formattedStartDate = getTimezoneFormattedTime(
			startTime,
			formData.timezone,
			!isEditMode,
			DATE_FORMAT,
		);

		const ordinalFormat = getTimezoneFormattedTime(
			startTime,
			formData.timezone,
			!isEditMode,
			ORDINAL_FORMAT,
		);

		const formattedDaysOfWeek = daysOfWeek
			?.map((day) => translateWeekday(t, day))
			.join(', ');
		switch (recurrenceType) {
			case 'daily':
				return t('pd_sched_daily', {
					date: formattedStartDate,
					time: formattedStartTime,
				});
			case 'monthly':
				return t('pd_sched_monthly', {
					date: formattedStartDate,
					ordinal: ordinalFormat,
					time: formattedStartTime,
				});
			case 'weekly':
				return t('pd_sched_weekly', {
					date: formattedStartDate,
					days: formattedDaysOfWeek ? `[${formattedDaysOfWeek}]` : '',
					time: formattedStartTime,
				});
			default:
				return t('pd_sched_default', {
					date: formattedStartDate,
					time: formattedStartTime,
				});
		}
	}, [formData, recurrenceType, isEditMode, timezoneInitialValue, t]);

	const endTimeText = useMemo((): string => {
		let endTime = formData?.endTime;
		if (recurrenceType !== recurrenceOptions.doesNotRepeat.value) {
			endTime =
				(formData?.recurrence?.endTime
					? dayjs(formData.recurrence.endTime).toISOString()
					: '') || '';

			if (!isEditMode && !endTime) {
				endTime = formData?.endTime || '';
			}
		}

		if (!endTime) {
			return '';
		}

		if (formData.timezone) {
			endTime = handleTimeConversion(
				endTime,
				timezoneInitialValue,
				formData?.timezone,
				!isEditMode,
			);
		}

		const formattedEndTime = getTimezoneFormattedTime(
			endTime,
			formData.timezone,
			!isEditMode,
			TIME_FORMAT,
		);

		const formattedEndDate = getTimezoneFormattedTime(
			endTime,
			formData.timezone,
			!isEditMode,
			DATE_FORMAT,
		);
		return t('pd_sched_end', {
			date: formattedEndDate,
			time: formattedEndTime,
		});
	}, [formData, recurrenceType, isEditMode, timezoneInitialValue, t]);

	return (
		<Modal
			title={
				<ModalTitle level={4}>
					{isEditMode ? t('pd_modal_edit') : t('pd_modal_new')}
				</ModalTitle>
			}
			centered
			open={isOpen}
			className="createDowntimeModal"
			onCancel={handleCancel}
			footer={null}
		>
			<Form<PlannedDowntimeFormData>
				name={initialValues.editMode ? 'edit-form' : 'create-form'}
				form={form}
				layout="vertical"
				className="createForm"
				onFinish={onFinish}
				onValuesChange={(): void => {
					setRecurrenceType(form.getFieldValue('recurrence')?.repeatType as string);
					setFormData(form.getFieldsValue());
				}}
				autoComplete="off"
			>
				<Form.Item label={t('pd_field_name')} name="name" rules={formValidationRules}>
					<Input placeholder={t('pd_name_placeholder')} />
				</Form.Item>
				<Form.Item
					label={t('pd_field_starts_from')}
					name="startTime"
					rules={formValidationRules}
					className={!isEmpty(startTimeText) ? 'formItemWithBullet' : ''}
					getValueProps={(value): any => ({
						value: value ? dayjs(value).tz(timezoneInitialValue) : undefined,
					})}
				>
					<DatePicker
						format={(date): string =>
							dayjs(date).tz(timezoneInitialValue).format(customFormat)
						}
						showTime
						renderExtraFooter={datePickerFooter}
						showNow={false}
						popupClassName="datePicker"
					/>
				</Form.Item>
				{!isEmpty(startTimeText) && (
					<div className="scheduleTimeInfoText">{startTimeText}</div>
				)}
				<Form.Item
					label={t('pd_field_repeats_every')}
					name={['recurrence', 'repeatType']}
					rules={formValidationRules}
				>
					<Select
						placeholder={t('pd_select_option')}
						options={recurrenceOptionWithSubmenu.map((option) => ({
							...option,
							label: translateRecurrenceType(t, option.value),
						}))}
					/>
				</Form.Item>
				{recurrenceType === recurrenceOptions.weekly.value && (
					<Form.Item
						label={t('pd_field_weekly')}
						name={['recurrence', 'repeatOn']}
						rules={formValidationRules}
					>
						<Select
							placeholder={t('pd_select_option')}
							mode="multiple"
							options={Object.values(recurrenceWeeklyOptions).map((option) => ({
								...option,
								label: translateWeekday(t, option.value),
							}))}
						/>
					</Form.Item>
				)}
				{recurrenceType &&
					recurrenceType !== recurrenceOptions.doesNotRepeat.value && (
						<Form.Item
							label={t('pd_field_duration')}
							name={['recurrence', 'duration']}
							rules={formValidationRules}
						>
							<Input
								addonAfter={
									<Select
										defaultValue="m"
										value={durationUnit}
										onChange={(value): void => {
											setDurationUnit(value);
										}}
									>
										<Select.Option value="m">{t('pd_unit_mins')}</Select.Option>
										<Select.Option value="h">{t('pd_unit_hours')}</Select.Option>
									</Select>
								}
								className="duration-input"
								type="number"
								placeholder={t('pd_duration_placeholder')}
								min={1}
								onWheel={(e): void => e.currentTarget.blur()}
							/>
						</Form.Item>
					)}
				<Form.Item
					label={t('pd_field_timezone')}
					name="timezone"
					rules={formValidationRules}
				>
					<Select
						options={timeZoneItems}
						placeholder={t('pd_timezone_placeholder')}
						showSearch
					/>
				</Form.Item>
				<Form.Item
					label={t('pd_field_ends_on')}
					name="endTime"
					required={recurrenceType === recurrenceOptions.doesNotRepeat.value}
					rules={[
						{
							required: recurrenceType === recurrenceOptions.doesNotRepeat.value,
						},
					]}
					className={!isEmpty(endTimeText) ? 'formItemWithBullet' : ''}
					getValueProps={(value): any => ({
						value: value ? dayjs(value).tz(timezoneInitialValue) : undefined,
					})}
				>
					<DatePicker
						format={(date): string =>
							dayjs(date).tz(timezoneInitialValue).format(customFormat)
						}
						showTime
						showNow={false}
						renderExtraFooter={datePickerFooter}
						popupClassName="datePicker"
					/>
				</Form.Item>
				{!isEmpty(endTimeText) && (
					<div className="scheduleTimeInfoText">{endTimeText}</div>
				)}
				<div>
					<div className="alert-rule-form">
						<Typography style={{ marginBottom: 8 }}>
							{t('pd_silence_alerts')}
						</Typography>
						<Typography style={{ marginBottom: 8 }} className="alert-rule-info">
							{t('pd_silence_all_hint')}
						</Typography>
					</div>
					<Form.Item noStyle shouldUpdate>
						<AlertRuleTags
							closable
							selectedTags={selectedTags}
							handleClose={handleClose}
						/>
					</Form.Item>
					<Form.Item name={alertRuleFormName}>
						<Select
							placeholder={t('pd_alerts_search_placeholder')}
							mode="multiple"
							status={isError ? 'error' : undefined}
							loading={isLoading}
							tagRender={noTagRenderer}
							onChange={handleChange}
							showSearch
							options={alertOptions}
							filterOption={(input, option): boolean =>
								(option?.label as string)?.toLowerCase()?.includes(input.toLowerCase())
							}
							notFoundContent={
								isLoading ? (
									<span>
										<Spin size="small" /> {t('pd_loading')}
									</span>
								) : (
									<span>{t('pd_no_alert')}</span>
								)
							}
						>
							{alertOptions?.map((option) => (
								<Select.Option key={option.value} value={option.value}>
									{option.label}
								</Select.Option>
							))}
						</Select>
					</Form.Item>
				</div>
				<Form.Item style={{ marginBottom: 0 }}>
					<ModalButtonWrapper>
						<Button
							key="submit"
							type="primary"
							htmlType="submit"
							icon={<CheckOutlined />}
							onClick={handleOk}
							loading={saveLoading || isLoading}
						>
							{isEditMode ? t('pd_btn_update') : t('pd_btn_add')}
						</Button>
					</ModalButtonWrapper>
				</Form.Item>
			</Form>
		</Modal>
	);
}
