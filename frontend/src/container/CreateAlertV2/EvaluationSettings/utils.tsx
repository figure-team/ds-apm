import * as Sentry from '@sentry/react';
import dayjs, { Dayjs } from 'dayjs';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';
import { rrulestr } from 'rrule';

import { ADVANCED_OPTIONS_TIME_UNIT_OPTIONS } from '../context/constants';
import { EvaluationWindowState } from '../context/types';
import { WEEKDAY_MAP } from './constants';
import { CumulativeWindowTimeframes, RollingWindowTimeframes } from './types';

// Extend dayjs with timezone plugins
dayjs.extend(utc);
dayjs.extend(timezone);

export const getEvaluationWindowTypeText = (
	windowType: 'rolling' | 'cumulative',
): string => {
	switch (windowType) {
		case 'rolling':
			return '롤링';
		case 'cumulative':
			return '누적';
		default:
			return '';
	}
};

export const getCumulativeWindowTimeframeText = (
	evaluationWindow: EvaluationWindowState,
): string => {
	switch (evaluationWindow.timeframe) {
		case CumulativeWindowTimeframes.CURRENT_HOUR:
			return `현재 시간, ${evaluationWindow.startingAt.number}분부터 (${evaluationWindow.startingAt.timezone})`;
		case CumulativeWindowTimeframes.CURRENT_DAY:
			return `현재 일, ${evaluationWindow.startingAt.time}부터 (${evaluationWindow.startingAt.timezone})`;
		case CumulativeWindowTimeframes.CURRENT_MONTH:
			return `현재 월, ${evaluationWindow.startingAt.number}일 ${evaluationWindow.startingAt.time}부터 (${evaluationWindow.startingAt.timezone})`;
		default:
			return '';
	}
};

export const getRollingWindowTimeframeText = (
	timeframe: RollingWindowTimeframes,
): string => {
	switch (timeframe) {
		case RollingWindowTimeframes.LAST_5_MINUTES:
			return '최근 5분';
		case RollingWindowTimeframes.LAST_10_MINUTES:
			return '최근 10분';
		case RollingWindowTimeframes.LAST_15_MINUTES:
			return '최근 15분';
		case RollingWindowTimeframes.LAST_30_MINUTES:
			return '최근 30분';
		case RollingWindowTimeframes.LAST_1_HOUR:
			return '최근 1시간';
		case RollingWindowTimeframes.LAST_2_HOURS:
			return '최근 2시간';
		case RollingWindowTimeframes.LAST_4_HOURS:
			return '최근 4시간';
		default:
			return '';
	}
};

export const getCustomRollingWindowTimeframeText = (
	evaluationWindow: EvaluationWindowState,
): string =>
	`최근 ${evaluationWindow.startingAt.number}${
		ADVANCED_OPTIONS_TIME_UNIT_OPTIONS.find(
			(option) => option.value === evaluationWindow.startingAt.unit,
		)?.label
	}`;

export const getTimeframeText = (
	evaluationWindow: EvaluationWindowState,
): string => {
	if (evaluationWindow.windowType === 'rolling') {
		if (evaluationWindow.timeframe === 'custom') {
			return getCustomRollingWindowTimeframeText(evaluationWindow);
		}
		return getRollingWindowTimeframeText(
			evaluationWindow.timeframe as RollingWindowTimeframes,
		);
	}
	return getCumulativeWindowTimeframeText(evaluationWindow);
};

export function buildAlertScheduleFromRRule(
	rruleString: string,
	date: Dayjs | null,
	startAt: string,
	maxOccurrences = 10,
): Date[] | null {
	try {
		if (!rruleString) {
			return null;
		}

		// Handle literal \n in string
		let finalRRuleString = rruleString.replace(/\\n/g, '\n');

		if (date) {
			const dt = dayjs(date);
			if (!dt.isValid()) {
				throw new Error('Invalid date provided');
			}

			const [hours = 0, minutes = 0, seconds = 0] = startAt.split(':').map(Number);

			const dtWithTime = dt
				.set('hour', hours)
				.set('minute', minutes)
				.set('second', seconds)
				.set('millisecond', 0);

			const dtStartStr = dtWithTime
				.toISOString()
				.replace(/[-:]/g, '')
				.replace(/\.\d{3}Z$/, 'Z');

			if (!/DTSTART/i.test(finalRRuleString)) {
				finalRRuleString = `DTSTART:${dtStartStr}\n${finalRRuleString}`;
			}
		}

		const rruleObj = rrulestr(finalRRuleString);
		const occurrences: Date[] = [];
		rruleObj.all((date, index) => {
			if (index >= maxOccurrences) {
				return false;
			}
			occurrences.push(date);
			return true;
		});

		return occurrences;
	} catch (error) {
		return null;
	}
}

function generateMonthlyOccurrences(
	targetDays: number[],
	hours: number,
	minutes: number,
	seconds: number,
	maxOccurrences: number,
): Date[] {
	const occurrences: Date[] = [];
	const currentMonth = dayjs().startOf('month');

	const currentDate = dayjs();

	const scanMonths = maxOccurrences + 12;
	for (let monthOffset = 0; monthOffset < scanMonths; monthOffset++) {
		const monthDate = currentMonth.add(monthOffset, 'month');
		targetDays.forEach((day) => {
			if (occurrences.length >= maxOccurrences) {
				return;
			}

			const daysInMonth = monthDate.daysInMonth();
			if (day <= daysInMonth) {
				const targetDate = monthDate
					.date(day)
					.hour(hours)
					.minute(minutes)
					.second(seconds);
				if (targetDate.isAfter(currentDate)) {
					occurrences.push(targetDate.toDate());
				}
			}
		});
	}

	return occurrences;
}

function generateWeeklyOccurrences(
	targetWeekdays: number[],
	hours: number,
	minutes: number,
	seconds: number,
	maxOccurrences: number,
): Date[] {
	const occurrences: Date[] = [];
	const currentWeek = dayjs().startOf('week');

	const currentDate = dayjs();

	for (let weekOffset = 0; weekOffset < maxOccurrences; weekOffset++) {
		const weekDate = currentWeek.add(weekOffset, 'week');
		targetWeekdays.forEach((weekday) => {
			if (occurrences.length >= maxOccurrences) {
				return;
			}

			const targetDate = weekDate
				.day(weekday)
				.hour(hours)
				.minute(minutes)
				.second(seconds);
			if (targetDate.isAfter(currentDate)) {
				occurrences.push(targetDate.toDate());
			}
		});
	}

	return occurrences;
}

export function generateDailyOccurrences(
	hours: number,
	minutes: number,
	seconds: number,
	maxOccurrences: number,
): Date[] {
	const occurrences: Date[] = [];
	const currentDate = dayjs();
	const currentTime =
		currentDate.hour() * 3600 + currentDate.minute() * 60 + currentDate.second();
	const targetTime = hours * 3600 + minutes * 60 + seconds;

	// Start from today if target time is after current time, otherwise start from tomorrow
	const startDayOffset = targetTime > currentTime ? 0 : 1;

	for (
		let dayOffset = startDayOffset;
		dayOffset < startDayOffset + maxOccurrences;
		dayOffset++
	) {
		const dayDate = currentDate.add(dayOffset, 'day');
		const targetDate = dayDate.hour(hours).minute(minutes).second(seconds);
		occurrences.push(targetDate.toDate());
	}

	return occurrences;
}

export function buildAlertScheduleFromCustomSchedule(
	repeatEvery: string,
	occurence: string[],
	startAt: string,
	maxOccurrences = 10,
): Date[] | null {
	try {
		const [hours = 0, minutes = 0, seconds = 0] = startAt.split(':').map(Number);
		let occurrences: Date[] = [];

		if (repeatEvery === 'month') {
			const targetDays = occurence
				.map((day) => parseInt(day, 10))
				.filter((day) => !Number.isNaN(day));
			occurrences = generateMonthlyOccurrences(
				targetDays,
				hours,
				minutes,
				seconds,
				maxOccurrences,
			);
		} else if (repeatEvery === 'week') {
			const targetWeekdays = occurence
				.map((day) => WEEKDAY_MAP[day.toLowerCase()])
				.filter((day) => day !== undefined);
			occurrences = generateWeeklyOccurrences(
				targetWeekdays,
				hours,
				minutes,
				seconds,
				maxOccurrences,
			);
		} else if (repeatEvery === 'day') {
			occurrences = generateDailyOccurrences(
				hours,
				minutes,
				seconds,
				maxOccurrences,
			);
		}

		occurrences.sort((a, b) => a.getTime() - b.getTime());
		return occurrences.slice(0, maxOccurrences);
	} catch (error) {
		Sentry.captureEvent({
			message: `Error building alert schedule from custom schedule: ${
				error instanceof Error ? error.message : 'Unknown error'
			}`,
			level: 'error',
		});
		return null;
	}
}

export function isValidRRule(rruleString: string): boolean {
	try {
		// normalize escaped \n
		const finalRRuleString = rruleString.replace(/\\n/g, '\n');
		rrulestr(finalRRuleString); // will throw if invalid
		return true;
	} catch {
		return false;
	}
}
