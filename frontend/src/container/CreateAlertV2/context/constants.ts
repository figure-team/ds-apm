import { Color } from '@signozhq/design-tokens';
import { UniversalYAxisUnit } from 'components/YAxisUnitSelector/types';
import { TIMEZONE_DATA } from 'container/CreateAlertV2/EvaluationSettings/constants';
import dayjs from 'dayjs';
import getRandomColor from 'lib/getRandomColor';
import { v4 } from 'uuid';

import {
	AdvancedOptionsState,
	AlertState,
	AlertThresholdMatchType,
	AlertThresholdOperator,
	AlertThresholdState,
	Algorithm,
	EvaluationWindowState,
	NotificationSettingsState,
	Seasonality,
	Threshold,
	TimeDuration,
} from './types';

export const INITIAL_ALERT_STATE: AlertState = {
	name: '',
	labels: {},
	annotations: {},
	yAxisUnit: undefined,
};

export const INITIAL_CRITICAL_THRESHOLD: Threshold = {
	id: v4(),
	label: 'critical',
	thresholdValue: 0,
	recoveryThresholdValue: null,
	unit: '',
	channels: [],
	color: Color.BG_SAKURA_500,
};

export const INITIAL_WARNING_THRESHOLD: Threshold = {
	id: v4(),
	label: 'warning',
	thresholdValue: 0,
	recoveryThresholdValue: null,
	unit: '',
	channels: [],
	color: Color.BG_AMBER_500,
};

export const INITIAL_INFO_THRESHOLD: Threshold = {
	id: v4(),
	label: 'info',
	thresholdValue: 0,
	recoveryThresholdValue: null,
	unit: '',
	channels: [],
	color: Color.BG_ROBIN_500,
};

export const INITIAL_ERROR_THRESHOLD: Threshold = {
	id: v4(),
	label: 'error',
	thresholdValue: 0,
	recoveryThresholdValue: null,
	unit: '',
	channels: [],
	color: Color.BG_CHERRY_500,
};

export const INITIAL_RANDOM_THRESHOLD: Threshold = {
	id: v4(),
	label: '',
	thresholdValue: 0,
	recoveryThresholdValue: null,
	unit: '',
	channels: [],
	color: getRandomColor(),
};

export const INITIAL_ALERT_THRESHOLD_STATE: AlertThresholdState = {
	selectedQuery: 'A',
	operator: AlertThresholdOperator.IS_ABOVE,
	matchType: AlertThresholdMatchType.AT_LEAST_ONCE,
	evaluationWindow: TimeDuration.FIVE_MINUTES,
	algorithm: Algorithm.STANDARD,
	seasonality: Seasonality.HOURLY,
	thresholds: [INITIAL_CRITICAL_THRESHOLD],
};

export const INITIAL_ADVANCED_OPTIONS_STATE: AdvancedOptionsState = {
	sendNotificationIfDataIsMissing: {
		toleranceLimit: 15,
		timeUnit: UniversalYAxisUnit.MINUTES,
		enabled: false,
	},
	enforceMinimumDatapoints: {
		minimumDatapoints: 0,
		enabled: false,
	},
	delayEvaluation: {
		delay: 5,
		timeUnit: UniversalYAxisUnit.MINUTES,
	},
	evaluationCadence: {
		mode: 'default',
		default: {
			value: 1,
			timeUnit: UniversalYAxisUnit.MINUTES,
		},
		custom: {
			repeatEvery: 'day',
			startAt: dayjs().format('HH:mm:ss'),
			occurence: [],
			timezone: TIMEZONE_DATA[0].value,
		},
		rrule: {
			date: dayjs(),
			startAt: dayjs().format('HH:mm:ss'),
			rrule: '',
		},
	},
};

export const INITIAL_EVALUATION_WINDOW_STATE: EvaluationWindowState = {
	windowType: 'rolling',
	timeframe: '5m0s',
	startingAt: {
		time: dayjs().format('HH:mm:ss'),
		number: '1',
		timezone: TIMEZONE_DATA[0].value,
		unit: UniversalYAxisUnit.MINUTES,
	},
};

export const THRESHOLD_OPERATOR_OPTIONS = [
	{ value: AlertThresholdOperator.IS_ABOVE, label: '초과' },
	{ value: AlertThresholdOperator.IS_BELOW, label: '미만' },
	{ value: AlertThresholdOperator.IS_EQUAL_TO, label: '같음' },
	{ value: AlertThresholdOperator.IS_NOT_EQUAL_TO, label: '같지 않음' },
];

export const ANOMALY_THRESHOLD_OPERATOR_OPTIONS = [
	{ value: AlertThresholdOperator.IS_ABOVE, label: '초과' },
	{ value: AlertThresholdOperator.IS_BELOW, label: '미만' },
	{ value: AlertThresholdOperator.ABOVE_BELOW, label: '초과/미만' },
];

export const THRESHOLD_MATCH_TYPE_OPTIONS = [
	{ value: AlertThresholdMatchType.AT_LEAST_ONCE, label: '한 번 이상' },
	{ value: AlertThresholdMatchType.ALL_THE_TIME, label: '항상' },
	{ value: AlertThresholdMatchType.ON_AVERAGE, label: '평균' },
	{ value: AlertThresholdMatchType.IN_TOTAL, label: '합계' },
	{ value: AlertThresholdMatchType.LAST, label: '마지막' },
];

export const ANOMALY_THRESHOLD_MATCH_TYPE_OPTIONS = [
	{ value: AlertThresholdMatchType.AT_LEAST_ONCE, label: '한 번 이상' },
	{ value: AlertThresholdMatchType.ALL_THE_TIME, label: '항상' },
];

export const ANOMALY_TIME_DURATION_OPTIONS = [
	{ value: TimeDuration.FIVE_MINUTES, label: '5분' },
	{ value: TimeDuration.TEN_MINUTES, label: '10분' },
	{ value: TimeDuration.FIFTEEN_MINUTES, label: '15분' },
	{ value: TimeDuration.ONE_HOUR, label: '1시간' },
	{ value: TimeDuration.THREE_HOURS, label: '3시간' },
	{ value: TimeDuration.FOUR_HOURS, label: '4시간' },
	{ value: TimeDuration.TWENTY_FOUR_HOURS, label: '24시간' },
];

export const ANOMALY_ALGORITHM_OPTIONS = [
	{ value: Algorithm.STANDARD, label: '표준' },
];

export const ANOMALY_SEASONALITY_OPTIONS = [
	{ value: Seasonality.HOURLY, label: '시간별' },
	{ value: Seasonality.DAILY, label: '일별' },
	{ value: Seasonality.WEEKLY, label: '주별' },
];

export const ADVANCED_OPTIONS_TIME_UNIT_OPTIONS = [
	{ value: UniversalYAxisUnit.SECONDS, label: '초' },
	{ value: UniversalYAxisUnit.MINUTES, label: '분' },
	{ value: UniversalYAxisUnit.HOURS, label: '시간' },
];

export const RE_NOTIFICATION_TIME_UNIT_OPTIONS = [
	{ value: UniversalYAxisUnit.MINUTES, label: '분' },
	{ value: UniversalYAxisUnit.HOURS, label: '시간' },
];

export const NOTIFICATION_MESSAGE_PLACEHOLDER =
	'정의된 메트릭(현재 값: {{$value}})이 임계값({{$threshold}})을 넘으면 발화하는 알림입니다';

export const RE_NOTIFICATION_CONDITION_OPTIONS = [
	{ value: 'firing', label: '발화 중' },
	{ value: 'nodata', label: '데이터 없음' },
];

export const INITIAL_NOTIFICATION_SETTINGS_STATE: NotificationSettingsState = {
	multipleNotifications: [],
	reNotification: {
		enabled: false,
		value: 30,
		unit: UniversalYAxisUnit.MINUTES,
		conditions: [],
	},
	description: NOTIFICATION_MESSAGE_PLACEHOLDER,
	routingPolicies: false,
};

export const INITIAL_CREATE_ALERT_STATE = {
	basic: INITIAL_ALERT_STATE,
	threshold: INITIAL_ALERT_THRESHOLD_STATE,
	advancedOptions: INITIAL_ADVANCED_OPTIONS_STATE,
	evaluationWindow: INITIAL_EVALUATION_WINDOW_STATE,
	notificationSettings: INITIAL_NOTIFICATION_SETTINGS_STATE,
};
