import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, DatePicker, Input, Select, Typography } from 'antd';
import classNames from 'classnames';
import { useCreateAlertState } from 'container/CreateAlertV2/context';
import { AdvancedOptionsState } from 'container/CreateAlertV2/context/types';
import dayjs from 'dayjs';
import { Code, Edit3Icon } from 'lucide-react';

import {
	EVALUATION_CADENCE_REPEAT_EVERY_MONTH_OPTIONS,
	EVALUATION_CADENCE_REPEAT_EVERY_OPTIONS,
	EVALUATION_CADENCE_REPEAT_EVERY_WEEK_OPTIONS,
	TIMEZONE_DATA,
} from '../constants';
import TimeInput from '../TimeInput';
import { IEvaluationCadenceDetailsProps } from '../types';
import {
	buildAlertScheduleFromCustomSchedule,
	buildAlertScheduleFromRRule,
	isValidRRule,
} from '../utils';
import { ScheduleList } from './EvaluationCadencePreview';

function EvaluationCadenceDetails({
	setIsOpen,
	setIsCustomScheduleButtonVisible,
}: IEvaluationCadenceDetailsProps): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { advancedOptions, setAdvancedOptions } = useCreateAlertState();
	const [evaluationCadence, setEvaluationCadence] = useState<
		AdvancedOptionsState['evaluationCadence']
	>({
		...advancedOptions.evaluationCadence,
		mode: 'custom',
		custom: {
			...advancedOptions.evaluationCadence.custom,
			startAt: dayjs().format('HH:mm:ss'),
		},
		rrule: {
			...advancedOptions.evaluationCadence.rrule,
			startAt: dayjs().format('HH:mm:ss'),
		},
	});

	const [searchTimezoneString, setSearchTimezoneString] = useState('');
	const [occurenceSearchString, setOccurenceSearchString] = useState('');
	const [repeatEverySearchString, setRepeatEverySearchString] = useState('');

	const tabs = [
		{
			label: 'Editor',
			icon: <Edit3Icon size={14} />,
			value: 'editor',
		},
		{
			label: 'RRule',
			icon: <Code size={14} />,
			value: 'rrule',
		},
	];
	const [activeTab, setActiveTab] = useState<'editor' | 'rrule'>(() =>
		evaluationCadence.mode === 'custom' ? 'editor' : 'rrule',
	);

	const occurenceOptions =
		evaluationCadence.custom.repeatEvery === 'week'
			? EVALUATION_CADENCE_REPEAT_EVERY_WEEK_OPTIONS
			: EVALUATION_CADENCE_REPEAT_EVERY_MONTH_OPTIONS;

	useEffect(() => {
		if (!evaluationCadence.custom.occurence.length) {
			const today = new Date();
			const dayOfWeek = today.getDay();
			const dayOfMonth = today.getDate();

			const occurence =
				evaluationCadence.custom.repeatEvery === 'week'
					? EVALUATION_CADENCE_REPEAT_EVERY_WEEK_OPTIONS[dayOfWeek].value
					: dayOfMonth.toString();

			setEvaluationCadence({
				...evaluationCadence,
				custom: {
					...evaluationCadence.custom,
					occurence: [occurence],
				},
			});
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [evaluationCadence.custom.repeatEvery]);

	const EditorView = (
		<div className="editor-view" data-testid="editor-view">
			<div className="select-group">
				<Typography.Text>{t('v2_cadence_repeat_every')}</Typography.Text>
				<Select
					options={EVALUATION_CADENCE_REPEAT_EVERY_OPTIONS}
					value={evaluationCadence.custom.repeatEvery || null}
					onChange={(value): void =>
						setEvaluationCadence({
							...evaluationCadence,
							custom: {
								...evaluationCadence.custom,
								repeatEvery: value,
								occurence: [],
							},
						})
					}
					placeholder={t('v2_cadence_repeat_every_placeholder')}
					showSearch
					searchValue={repeatEverySearchString}
					onSearch={setRepeatEverySearchString}
				/>
			</div>
			{evaluationCadence.custom.repeatEvery !== 'day' && (
				<div className="select-group">
					<Typography.Text>{t('v2_cadence_on_days')}</Typography.Text>
					<Select
						options={occurenceOptions}
						value={evaluationCadence.custom.occurence || null}
						mode="multiple"
						onChange={(value): void =>
							setEvaluationCadence({
								...evaluationCadence,
								custom: {
									...evaluationCadence.custom,
									occurence: value,
								},
							})
						}
						placeholder={t('v2_cadence_on_days_placeholder')}
						showSearch
						searchValue={occurenceSearchString}
						onSearch={setOccurenceSearchString}
					/>
				</div>
			)}
			<div className="select-group">
				<Typography.Text>{t('v2_cadence_at')}</Typography.Text>
				<TimeInput
					value={evaluationCadence.custom.startAt}
					onChange={(value): void =>
						setEvaluationCadence({
							...evaluationCadence,
							custom: {
								...evaluationCadence.custom,
								startAt: value,
							},
						})
					}
				/>
			</div>
			<div className="select-group">
				<Typography.Text>{t('v2_cadence_timezone')}</Typography.Text>
				<Select
					options={TIMEZONE_DATA}
					value={evaluationCadence.custom.timezone || null}
					onChange={(value): void =>
						setEvaluationCadence({
							...evaluationCadence,
							custom: {
								...evaluationCadence.custom,
								timezone: value,
							},
						})
					}
					placeholder={t('v2_cadence_timezone_placeholder')}
					onSearch={setSearchTimezoneString}
					searchValue={searchTimezoneString}
					showSearch
				/>
			</div>
		</div>
	);

	const RRuleView = (
		<div className="rrule-view" data-testid="rrule-view">
			<div className="select-group">
				<Typography.Text>{t('v2_cadence_starting_on')}</Typography.Text>
				<DatePicker
					value={evaluationCadence.rrule.date}
					onChange={(value): void =>
						setEvaluationCadence({
							...evaluationCadence,
							rrule: {
								...evaluationCadence.rrule,
								date: value,
							},
						})
					}
					placeholder={t('v2_cadence_starting_on_placeholder')}
				/>
			</div>
			<div className="select-group">
				<Typography.Text>{t('v2_cadence_at')}</Typography.Text>
				<TimeInput
					value={evaluationCadence.rrule.startAt}
					onChange={(value): void =>
						setEvaluationCadence({
							...evaluationCadence,
							rrule: {
								...evaluationCadence.rrule,
								startAt: value,
							},
						})
					}
				/>
			</div>
			<Input.TextArea
				value={evaluationCadence.rrule.rrule}
				placeholder={t('v2_cadence_rrule_placeholder')}
				onChange={(value): void =>
					setEvaluationCadence({
						...evaluationCadence,
						rrule: {
							...evaluationCadence.rrule,
							rrule: value.target.value,
						},
					})
				}
			/>
		</div>
	);

	const handleDiscard = (): void => {
		setIsOpen(false);
		setIsCustomScheduleButtonVisible(true);
	};

	const handleSaveCustomSchedule = (): void => {
		setAdvancedOptions({
			type: 'SET_EVALUATION_CADENCE',
			payload: {
				...advancedOptions.evaluationCadence,
				custom: evaluationCadence.custom,
				rrule: evaluationCadence.rrule,
			},
		});
		setAdvancedOptions({
			type: 'SET_EVALUATION_CADENCE_MODE',
			payload: evaluationCadence.mode,
		});
		setIsOpen(false);
	};

	const disableSaveButton = useMemo(() => {
		if (activeTab === 'editor') {
			if (evaluationCadence.custom.repeatEvery === 'day') {
				return (
					!evaluationCadence.custom.repeatEvery ||
					!evaluationCadence.custom.startAt ||
					!evaluationCadence.custom.timezone
				);
			}
			return (
				!evaluationCadence.custom.repeatEvery ||
				!evaluationCadence.custom.occurence.length ||
				!evaluationCadence.custom.startAt ||
				!evaluationCadence.custom.timezone
			);
		}
		return (
			!evaluationCadence.rrule.rrule ||
			!evaluationCadence.rrule.date ||
			!evaluationCadence.rrule.startAt ||
			!isValidRRule(evaluationCadence.rrule.rrule)
		);
	}, [evaluationCadence, activeTab]);

	const schedule = useMemo(() => {
		if (activeTab === 'rrule') {
			return buildAlertScheduleFromRRule(
				evaluationCadence.rrule.rrule,
				evaluationCadence.rrule.date,
				evaluationCadence.rrule.startAt,
				15,
			);
		}
		return buildAlertScheduleFromCustomSchedule(
			evaluationCadence.custom.repeatEvery,
			evaluationCadence.custom.occurence,
			evaluationCadence.custom.startAt,
			15,
		);
	}, [evaluationCadence, activeTab]);

	const handleChangeTab = (tab: 'editor' | 'rrule'): void => {
		setActiveTab(tab);
		const mode = tab === 'editor' ? 'custom' : 'rrule';
		setEvaluationCadence({
			...evaluationCadence,
			mode,
		});
	};

	return (
		<div className="evaluation-cadence-details">
			<Typography.Text className="evaluation-cadence-details-title">
				{t('v2_add_custom_schedule')}
			</Typography.Text>
			<div className="evaluation-cadence-details-content">
				<div className="evaluation-cadence-details-content-row">
					<div className="query-section-tabs">
						<div className="query-section-query-actions">
							{tabs.map((tab) => (
								<Button
									key={tab.value}
									className={classNames('list-view-tab', 'explorer-view-option', {
										'active-tab': activeTab === tab.value,
									})}
									onClick={(): void => {
										handleChangeTab(tab.value as 'editor' | 'rrule');
									}}
								>
									{tab.icon}
									{tab.label}
								</Button>
							))}
						</div>
					</div>
					{activeTab === 'editor' && EditorView}
					{activeTab === 'rrule' && RRuleView}
					<div className="buttons-row">
						<Button type="default" onClick={handleDiscard}>
							{t('v2_discard_schedule')}
						</Button>
						<Button
							type="primary"
							onClick={handleSaveCustomSchedule}
							disabled={disableSaveButton}
						>
							{t('v2_save_custom_schedule')}
						</Button>
					</div>
				</div>
				<div className="evaluation-cadence-details-content-row">
					<ScheduleList
						schedule={schedule}
						currentTimezone={evaluationCadence.custom.timezone}
					/>
				</div>
			</div>
		</div>
	);
}

export default EvaluationCadenceDetails;
