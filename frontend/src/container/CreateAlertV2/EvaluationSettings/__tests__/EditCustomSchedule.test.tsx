import { fireEvent, render, screen } from '@testing-library/react';
import * as alertState from 'container/CreateAlertV2/context';
import { INITIAL_ADVANCED_OPTIONS_STATE } from 'container/CreateAlertV2/context/constants';

import { TIMEZONE_DATA } from '../constants';
import EditCustomSchedule from '../EvaluationCadence/EditCustomSchedule';
import { createMockAlertContextState } from './testUtils';

const mockSetAdvancedOptions = jest.fn();
jest.spyOn(alertState, 'useCreateAlertState').mockReturnValue(
	createMockAlertContextState({
		setAdvancedOptions: mockSetAdvancedOptions,
	}),
);

const mockSetIsEvaluationCadenceDetailsVisible = jest.fn();
const mockSetIsPreviewVisible = jest.fn();

const EDIT_CUSTOM_SCHEDULE_TEST_ID = '.edit-custom-schedule';

describe('EditCustomSchedule', () => {
	it('should render the correct display text for custom mode with daily occurrence', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE.evaluationCadence,
						mode: 'custom',
						custom: {
							repeatEvery: 'day',
							startAt: '00:00:00',
							occurence: [],
							timezone: TIMEZONE_DATA[0].value,
						},
					},
				},
			}),
		);
		render(
			<EditCustomSchedule
				setIsEvaluationCadenceDetailsVisible={
					mockSetIsEvaluationCadenceDetailsVisible
				}
				setIsPreviewVisible={mockSetIsPreviewVisible}
			/>,
		);

		// Use textContent to verify the complete text across multiple Typography components
		const container = screen
			.getByText('v2_edit_custom_schedule_every')
			.closest(EDIT_CUSTOM_SCHEDULE_TEST_ID);
		expect(container).toHaveTextContent('v2_edit_custom_schedule_everyDayv2_edit_custom_schedule_at00:00:00');
	});

	it('should render the correct display text for custom mode with weekly occurrence', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE.evaluationCadence,
						mode: 'custom',
						custom: {
							repeatEvery: 'week',
							startAt: '00:00:00',
							occurence: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'],
							timezone: TIMEZONE_DATA[0].value,
						},
					},
				},
			}),
		);

		render(
			<EditCustomSchedule
				setIsEvaluationCadenceDetailsVisible={
					mockSetIsEvaluationCadenceDetailsVisible
				}
				setIsPreviewVisible={mockSetIsPreviewVisible}
			/>,
		);

		const container = screen
			.getByText('v2_edit_custom_schedule_every')
			.closest(EDIT_CUSTOM_SCHEDULE_TEST_ID);
		expect(container).toHaveTextContent(
			'v2_edit_custom_schedule_everyWeekv2_edit_custom_schedule_onMonday, Tuesday, Wednesday, Thursday, Fridayv2_edit_custom_schedule_at00:00:00',
		);
	});

	it('should render the correct display text for custom mode with monthly occurrence', () => {
		jest.spyOn(alertState, 'useCreateAlertState').mockReturnValueOnce(
			createMockAlertContextState({
				advancedOptions: {
					...INITIAL_ADVANCED_OPTIONS_STATE,
					evaluationCadence: {
						...INITIAL_ADVANCED_OPTIONS_STATE.evaluationCadence,
						mode: 'custom',
						custom: {
							repeatEvery: 'month',
							startAt: '00:00:00',
							occurence: ['1'],
							timezone: TIMEZONE_DATA[0].value,
						},
					},
				},
			}),
		);

		render(
			<EditCustomSchedule
				setIsEvaluationCadenceDetailsVisible={
					mockSetIsEvaluationCadenceDetailsVisible
				}
				setIsPreviewVisible={mockSetIsPreviewVisible}
			/>,
		);

		const container = screen
			.getByText('v2_edit_custom_schedule_every')
			.closest(EDIT_CUSTOM_SCHEDULE_TEST_ID);
		expect(container).toHaveTextContent('v2_edit_custom_schedule_everyMonthv2_edit_custom_schedule_on1v2_edit_custom_schedule_at00:00:00');
	});

	it('edit custom schedule action works correctly', () => {
		render(
			<EditCustomSchedule
				setIsEvaluationCadenceDetailsVisible={
					mockSetIsEvaluationCadenceDetailsVisible
				}
				setIsPreviewVisible={mockSetIsPreviewVisible}
			/>,
		);

		fireEvent.click(screen.getByText('v2_edit_custom_schedule_btn'));
		expect(mockSetIsEvaluationCadenceDetailsVisible).toHaveBeenCalledWith(true);
		expect(mockSetIsPreviewVisible).not.toHaveBeenCalled();
	});

	it('preview custom schedule action works correctly', () => {
		render(
			<EditCustomSchedule
				setIsEvaluationCadenceDetailsVisible={
					mockSetIsEvaluationCadenceDetailsVisible
				}
				setIsPreviewVisible={mockSetIsPreviewVisible}
			/>,
		);

		fireEvent.click(screen.getByText('v2_schedule_preview_btn'));
		expect(mockSetIsPreviewVisible).toHaveBeenCalledWith(true);
		expect(mockSetIsEvaluationCadenceDetailsVisible).not.toHaveBeenCalled();
	});
});
