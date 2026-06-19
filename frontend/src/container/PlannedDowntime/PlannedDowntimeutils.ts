import { UseMutateAsyncFunction } from 'react-query';
import type { NotificationInstance } from 'antd/es/notification/interface';
import type { DefaultOptionType } from 'antd/es/select';
import { convertToApiError } from 'api/ErrorResponseHandlerForGeneratedAPIs';
import type {
	DeleteDowntimeScheduleByIDPathParameters,
	RenderErrorResponseDTO,
	RuletypesPlannedMaintenanceDTO,
	RuletypesRecurrenceDTO,
} from 'api/generated/services/sigNoz.schemas';
import type { ErrorType } from 'api/generatedAPIInstance';
import { AxiosError } from 'axios';
import { DATE_TIME_FORMATS } from 'constants/dateTimeFormats';
import dayjs from 'dayjs';
import { TFunction } from 'i18next';
import { isEmpty, isEqual } from 'lodash-es';
import APIError from 'types/api/error';

type DateTimeString = string | null | undefined;

export const translateRecurrenceType = (
	t: TFunction,
	value?: string | null,
): string => {
	switch (value) {
		case recurrenceOptions.daily.value:
			return t('pd_recur_daily');
		case recurrenceOptions.weekly.value:
			return t('pd_recur_weekly');
		case recurrenceOptions.monthly.value:
			return t('pd_recur_monthly');
		case recurrenceOptions.doesNotRepeat.value:
			return t('pd_recur_does_not_repeat');
		default:
			return value || '';
	}
};

export const translateWeekday = (t: TFunction, value: string): string => {
	const key = `pd_day_${value}`;
	const translated = t(key);
	return translated === key ? value : translated;
};

export const getDuration = (
	t: TFunction,
	startTime: DateTimeString,
	endTime: DateTimeString,
): string => {
	if (!startTime || !endTime) {
		return t('pd_na');
	}

	const start = dayjs(startTime);
	const end = dayjs(endTime);
	const durationMs = end.diff(start);

	const minutes = Math.floor(durationMs / (1000 * 60));
	const hours = Math.floor(durationMs / (1000 * 60 * 60));

	if (minutes < 60) {
		return t('pd_duration_min', { count: minutes });
	}
	return t('pd_duration_hours', { count: hours });
};

export const formatDateTime = (
	t: TFunction,
	dateTimeString?: string | null,
): string => {
	if (!dateTimeString) {
		return t('pd_na');
	}

	return dayjs(dateTimeString.slice(0, 19)).format(
		DATE_TIME_FORMATS.MONTH_DATETIME,
	);
};

export const getAlertOptionsFromIds = (
	alertIds: string[],
	alertOptions: DefaultOptionType[],
): DefaultOptionType[] =>
	alertOptions.filter(
		(alert) =>
			alert !== undefined &&
			alert.value &&
			alertIds?.includes(alert.value as string),
	);

export const recurrenceInfo = (
	t: TFunction,
	recurrence?: RuletypesRecurrenceDTO | null,
): string => {
	if (!recurrence) {
		return t('pd_recur_no');
	}

	const { startTime, duration, repeatOn, repeatType, endTime } = recurrence;

	const formattedStartTime = startTime
		? formatDateTime(t, dayjs(startTime).toISOString())
		: '';
	const formattedEndTime = endTime
		? t('pd_recurrence_to', {
				end: formatDateTime(t, dayjs(endTime).toISOString()),
			})
		: '';
	const weeklyRepeatString = repeatOn
		? t('pd_recurrence_on', {
				days: repeatOn.map((day) => translateWeekday(t, day)).join(', '),
			})
		: '';
	const durationString = duration
		? t('pd_recurrence_duration', { duration })
		: '';

	return t('pd_recurrence_info', {
		repeatType: translateRecurrenceType(t, repeatType),
		weekly: weeklyRepeatString,
		start: formattedStartTime,
		end: formattedEndTime,
		duration: durationString,
	});
};

export const defautlInitialValues: Partial<
	RuletypesPlannedMaintenanceDTO & { editMode: boolean }
> = {
	name: '',
	description: '',
	schedule: {
		timezone: '',
		endTime: undefined,
		recurrence: undefined,
		startTime: undefined,
	},
	alertIds: [],
	createdAt: undefined,
	createdBy: undefined,
	editMode: false,
};

type DeleteDowntimeScheduleProps = {
	deleteDowntimeScheduleAsync: UseMutateAsyncFunction<
		void,
		ErrorType<RenderErrorResponseDTO>,
		{ pathParams: DeleteDowntimeScheduleByIDPathParameters }
	>;
	notifications: NotificationInstance;
	showErrorModal: (error: APIError) => void;
	refetchAllSchedules: VoidFunction;
	deleteId?: string;
	hideDeleteDowntimeScheduleModal: () => void;
	clearSearch: () => void;
	t: TFunction;
};

export const deleteDowntimeHandler = ({
	deleteDowntimeScheduleAsync,
	refetchAllSchedules,
	deleteId,
	hideDeleteDowntimeScheduleModal,
	clearSearch,
	notifications,
	showErrorModal,
	t,
}: DeleteDowntimeScheduleProps): void => {
	if (!deleteId) {
		console.error('Unable to delete, please provide correct deleteId');
		notifications.error({ message: t('pd_something_wrong').toString() });
	} else {
		deleteDowntimeScheduleAsync(
			{ pathParams: { id: String(deleteId) } },
			{
				onSuccess: () => {
					hideDeleteDowntimeScheduleModal();
					clearSearch();
					notifications.success({
						message: t('pd_schedule_deleted').toString(),
					});
					refetchAllSchedules();
				},
				onError: (err) => {
					showErrorModal(
						convertToApiError(err as AxiosError<RenderErrorResponseDTO>) as APIError,
					);
				},
			},
		);
	}
};

export const recurrenceOptions = {
	doesNotRepeat: {
		label: 'Does not repeat',
		value: 'does-not-repeat',
	},
	daily: { label: 'Daily', value: 'daily' },
	weekly: { label: 'Weekly', value: 'weekly' },
	monthly: { label: 'Monthly', value: 'monthly' },
};

export const recurrenceWeeklyOptions = {
	monday: { label: 'Monday', value: 'monday' },
	tuesday: { label: 'Tuesday', value: 'tuesday' },
	wednesday: { label: 'Wednesday', value: 'wednesday' },
	thursday: { label: 'Thursday', value: 'thursday' },
	friday: { label: 'Friday', value: 'friday' },
	saturday: { label: 'Saturday', value: 'saturday' },
	sunday: { label: 'Sunday', value: 'sunday' },
};
interface DurationInfo {
	value: number;
	unit: string;
}

export function getDurationInfo(
	durationString: string | undefined | null,
): DurationInfo | null {
	if (!durationString) {
		return null;
	}

	// Regular expressions to extract hours, minutes
	const hoursRegex = /(\d+)h/;
	const minutesRegex = /(\d+)m/;

	// Extract hours, minutes from the duration string
	const hoursMatch = durationString.match(hoursRegex);
	const minutesMatch = durationString.match(minutesRegex);

	// Convert extracted values to integers, defaulting to 0 if not found
	const hours = hoursMatch ? parseInt(hoursMatch[1], 10) : 0;
	const minutes = minutesMatch ? parseInt(minutesMatch[1], 10) : 0;

	// If there are no minutes and only hours, return the hours
	if (hours > 0 && minutes === 0) {
		return { value: hours, unit: 'h' };
	}

	// Otherwise, calculate the total duration in minutes
	const totalMinutes = hours * 60 + minutes;
	return { value: totalMinutes, unit: 'm' };
}

export interface Option {
	label: string;
	value: string;
}

export const recurrenceOptionWithSubmenu: Option[] = [
	recurrenceOptions.doesNotRepeat,
	recurrenceOptions.daily,
	recurrenceOptions.weekly,
	recurrenceOptions.monthly,
];

export const getRecurrenceOptionFromValue = (
	value?: string | Option | null,
): Option | null | undefined => {
	if (!value) {
		return null;
	}
	if (typeof value === 'string') {
		return Object.values(recurrenceOptions).find(
			(option) => option.value === value,
		);
	}
	return value;
};

export const getEndTime = ({
	kind,
	schedule,
}: Partial<
	RuletypesPlannedMaintenanceDTO & {
		editMode: boolean;
	}
>): string | dayjs.Dayjs => {
	if (kind === 'fixed') {
		return schedule?.endTime ? dayjs(schedule.endTime).toISOString() : '';
	}

	return schedule?.recurrence?.endTime
		? dayjs(schedule.recurrence.endTime).toISOString()
		: '';
};

export const isScheduleRecurring = (
	schedule?: RuletypesPlannedMaintenanceDTO['schedule'] | null,
): boolean => (schedule ? !isEmpty(schedule?.recurrence) : false);

function convertUtcOffsetToTimezoneOffset(offsetMinutes: number): string {
	const sign = offsetMinutes >= 0 ? '+' : '-';
	const absOffset = Math.abs(offsetMinutes);
	const hours = String(Math.floor(absOffset / 60)).padStart(2, '0');
	const minutes = String(absOffset % 60).padStart(2, '0');
	return `${sign}${hours}:${minutes}`;
}

export function formatWithTimezone(
	dateValue?: string | dayjs.Dayjs,
	timezone?: string,
): string {
	const parsedDate =
		typeof dateValue === 'string' ? dateValue : dateValue?.format();

	// Get the target timezone offset
	const targetOffset = convertUtcOffsetToTimezoneOffset(
		dayjs(dateValue).tz(timezone).utcOffset(),
	);

	return `${parsedDate?.substring(0, 19)}${targetOffset}`;
}

export function handleTimeConversion(
	dateValue: string | dayjs.Dayjs,
	timezoneInit?: string,
	timezone?: string,
	shouldKeepLocalTime?: boolean,
): string {
	const timezoneChanged = !isEqual(timezoneInit, timezone);
	const initialTime = dayjs(dateValue).tz(timezoneInit);

	const formattedTime = formatWithTimezone(initialTime, timezone);
	return timezoneChanged
		? formattedTime
		: dayjs(dateValue).tz(timezone, shouldKeepLocalTime).format();
}
