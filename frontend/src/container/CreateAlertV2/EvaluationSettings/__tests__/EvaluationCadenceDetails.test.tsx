import { fireEvent, render, screen } from '@testing-library/react';
import * as alertState from 'container/CreateAlertV2/context';
import { INITIAL_ADVANCED_OPTIONS_STATE } from 'container/CreateAlertV2/context/constants';
import { AdvancedOptionsState } from 'container/CreateAlertV2/context/types';

import EvaluationCadenceDetails from '../EvaluationCadence/EvaluationCadenceDetails';
import { createMockAlertContextState } from './testUtils';

const ENTER_RRULE_PLACEHOLDER = 'v2_cadence_rrule_placeholder';

jest.mock('dayjs', () => {
	const actualDayjs = jest.requireActual('dayjs');
	const mockDayjs = (date?: any): any => {
		if (date) {
			return actualDayjs(date);
		}
		// 21 Jan 2025
		return actualDayjs('2025-01-21T16:31:36.982Z');
	};
	Object.keys(actualDayjs).forEach((key) => {
		if (typeof (actualDayjs as any)[key] === 'function') {
			(mockDayjs as any)[key] = (actualDayjs as any)[key];
		}
	});
	(mockDayjs as any).tz = {
		guess: (): string => 'Asia/Saigon',
	};
	return mockDayjs;
});

const INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE: AdvancedOptionsState =
	{
		...INITIAL_ADVANCED_OPTIONS_STATE,
		evaluationCadence: {
			...INITIAL_ADVANCED_OPTIONS_STATE.evaluationCadence,
			mode: 'custom',
		},
	};

const mockSetAdvancedOptions = jest.fn();
jest.spyOn(alertState, 'useCreateAlertState').mockReturnValue(
	createMockAlertContextState({
		advancedOptions: INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE,
		setAdvancedOptions: mockSetAdvancedOptions,
	}),
);

const mockSetIsOpen = jest.fn();
const mockSetIsCustomScheduleButtonVisible = jest.fn();

const SCHEDULE_PREVIEW_TEST_ID = 'schedule-preview';
const NO_SCHEDULE_TEST_ID = 'no-schedule';
const EDITOR_VIEW_TEST_ID = 'editor-view';
const RULE_VIEW_TEST_ID = 'rrule-view';
const SAVE_CUSTOM_SCHEDULE_TEXT = 'v2_save_custom_schedule';

describe('EvaluationCadenceDetails', () => {
	it('should render the evaluation cadence details component with editor mode in daily occurence by default', () => {
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);
		expect(screen.getByText('v2_add_custom_schedule')).toBeInTheDocument();

		expect(screen.getByTestId(EDITOR_VIEW_TEST_ID)).toBeInTheDocument();
		expect(screen.queryByTestId('rrule-view')).not.toBeInTheDocument();

		expect(screen.getByText('v2_cadence_repeat_every')).toBeInTheDocument();
		expect(screen.getByText('v2_cadence_at')).toBeInTheDocument();
		expect(screen.getByText('v2_cadence_timezone')).toBeInTheDocument();

		expect(screen.getByTestId(SCHEDULE_PREVIEW_TEST_ID)).toBeInTheDocument();

		expect(screen.getByText('v2_discard_schedule')).toBeInTheDocument();
		expect(screen.getByText(SAVE_CUSTOM_SCHEDULE_TEXT)).toBeInTheDocument();
	});

	it('when switching to rrule mode, the rrule view should be rendered with no schedule preview', () => {
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);
		fireEvent.click(screen.getByText('RRule'));
		expect(screen.getByTestId(RULE_VIEW_TEST_ID)).toBeInTheDocument();

		expect(
			screen.queryByTestId(SCHEDULE_PREVIEW_TEST_ID),
		).not.toBeInTheDocument();
		expect(screen.getByTestId(NO_SCHEDULE_TEST_ID)).toBeInTheDocument();

		expect(screen.getByText('v2_cadence_starting_on')).toBeInTheDocument();
		expect(screen.getByText('v2_cadence_at')).toBeInTheDocument();
		expect(
			screen.getByPlaceholderText(ENTER_RRULE_PLACEHOLDER),
		).toBeInTheDocument();

		expect(screen.getByText('v2_discard_schedule')).toBeInTheDocument();
		expect(screen.getByText(SAVE_CUSTOM_SCHEDULE_TEXT)).toBeInTheDocument();
	});

	it('when showing weekly occurence, the occurence options should be rendered', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence,
						custom: {
							...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence
								.custom,
							repeatEvery: 'week',
						},
					},
				},
			}),
		);
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);

		// Verify that the "ON DAY(S)" section is rendered for weekly occurrence
		expect(screen.getByText('v2_cadence_on_days')).toBeInTheDocument();

		// Verify that the schedule preview is shown as today is selected by default
		expect(screen.getByTestId(SCHEDULE_PREVIEW_TEST_ID)).toBeInTheDocument();
		expect(screen.queryByTestId(NO_SCHEDULE_TEST_ID)).not.toBeInTheDocument();
	});

	it('render schedule preview in weekly occurence when days are selected', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence,
						custom: {
							...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence
								.custom,
							repeatEvery: 'week',
							occurence: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'],
						},
					},
				},
			}),
		);
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);

		// Verify that the schedule preview is shown because days are selected
		expect(screen.getByTestId(SCHEDULE_PREVIEW_TEST_ID)).toBeInTheDocument();
		expect(screen.queryByTestId(NO_SCHEDULE_TEST_ID)).not.toBeInTheDocument();
	});

	it('when showing monthly occurence, the occurence options should be rendered', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence,
						custom: {
							...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence
								.custom,
							repeatEvery: 'month',
						},
					},
				},
			}),
		);
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);

		// Verify that the "ON DAY(S)" section is rendered for monthly occurrence
		expect(screen.getByText('v2_cadence_on_days')).toBeInTheDocument();

		// Verify that the schedule preview is  shown as today is selected by default
		expect(screen.getByTestId(SCHEDULE_PREVIEW_TEST_ID)).toBeInTheDocument();
		expect(screen.queryByTestId(NO_SCHEDULE_TEST_ID)).not.toBeInTheDocument();
	});

	it('render schedule preview in monthly occurence when days are selected', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence,
						custom: {
							...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence
								.custom,
							repeatEvery: 'month',
							occurence: ['1'],
						},
					},
				},
			}),
		);
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);

		// Verify that the schedule preview is shown because days are selected
		expect(screen.getByTestId(SCHEDULE_PREVIEW_TEST_ID)).toBeInTheDocument();
		expect(screen.queryByTestId(NO_SCHEDULE_TEST_ID)).not.toBeInTheDocument();
	});

	it('discard action works correctly', () => {
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);
		fireEvent.click(screen.getByText('v2_discard_schedule'));
		expect(mockSetIsOpen).toHaveBeenCalledWith(false);
		expect(mockSetIsCustomScheduleButtonVisible).toHaveBeenCalledWith(true);
	});

	it('save custom schedule action works correctly', () => {
		render(
			<EvaluationCadenceDetails
				isOpen
				setIsOpen={mockSetIsOpen}
				setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
			/>,
		);
		fireEvent.click(screen.getByText(SAVE_CUSTOM_SCHEDULE_TEXT));
		expect(mockSetAdvancedOptions).toHaveBeenCalledTimes(2);
		expect(mockSetAdvancedOptions).toHaveBeenCalledWith({
			type: 'SET_EVALUATION_CADENCE',
			payload: {
				...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence,
				custom: {
					...INITIAL_ADVANCED_OPTIONS_STATE_WITH_CUSTOM_SCHEDULE.evaluationCadence
						.custom,
					// today selected by default
					occurence: [new Date().getDate().toString()],
				},
			},
		});
		expect(mockSetAdvancedOptions).toHaveBeenCalledWith({
			type: 'SET_EVALUATION_CADENCE_MODE',
			payload: 'custom',
		});
	});

	describe('alert context mock state verification', () => {
		it('should set the evaluation cadence tab to rrule from custom', () => {
			render(
				<EvaluationCadenceDetails
					isOpen
					setIsOpen={mockSetIsOpen}
					setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
				/>,
			);

			// Switch to RRule tab
			fireEvent.click(screen.getByText('RRule'));
			expect(screen.getByTestId(RULE_VIEW_TEST_ID)).toBeInTheDocument();
			expect(screen.queryByTestId(EDITOR_VIEW_TEST_ID)).not.toBeInTheDocument();

			// Type in the text box
			expect(screen.getByPlaceholderText(ENTER_RRULE_PLACEHOLDER)).toHaveValue('');
			fireEvent.change(screen.getByPlaceholderText(ENTER_RRULE_PLACEHOLDER), {
				target: { value: 'RRULE:FREQ=DAILY' },
			});
			// Ensure text box content is updated
			expect(screen.getByPlaceholderText(ENTER_RRULE_PLACEHOLDER)).toHaveValue(
				'RRULE:FREQ=DAILY',
			);
		});

		it('ensure rrule content is not modified by previous test', () => {
			render(
				<EvaluationCadenceDetails
					isOpen
					setIsOpen={mockSetIsOpen}
					setIsCustomScheduleButtonVisible={mockSetIsCustomScheduleButtonVisible}
				/>,
			);

			// Switch to RRule tab
			fireEvent.click(screen.getByText('RRule'));
			expect(screen.getByTestId(RULE_VIEW_TEST_ID)).toBeInTheDocument();
			expect(screen.queryByTestId(EDITOR_VIEW_TEST_ID)).not.toBeInTheDocument();

			// Verify text box content
			expect(screen.getByPlaceholderText(ENTER_RRULE_PLACEHOLDER)).toHaveValue('');
		});
	});
});
