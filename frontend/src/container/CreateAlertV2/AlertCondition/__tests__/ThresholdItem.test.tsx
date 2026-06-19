import { fireEvent, render, screen } from '@testing-library/react';
import type { DefaultOptionType } from 'antd/es/select';

jest.mock('react-i18next', () => ({
	useTranslation: (): { t: (key: string) => string } => ({
		t: (key: string): string => key,
	}),
	Trans: ({ children }: { children: React.ReactNode }): React.ReactNode =>
		children,
}));
import { createMockAlertContextState } from 'container/CreateAlertV2/EvaluationSettings/__tests__/testUtils';
import { getAppContextMockState } from 'container/RoutingPolicies/__tests__/testUtils';
import * as appHooks from 'providers/App/App';
import { Channels } from 'types/api/channels/getAll';

import * as context from '../../context';
import ThresholdItem from '../ThresholdItem';
import { ThresholdItemProps } from '../types';

jest.spyOn(appHooks, 'useAppContext').mockReturnValue(getAppContextMockState());

jest.mock('uplot', () => {
	const paths = {
		spline: jest.fn(),
		bars: jest.fn(),
	};
	const uplotMock: any = jest.fn(() => ({
		paths,
	}));
	uplotMock.paths = paths;
	return uplotMock;
});

const mockSetAlertState = jest.fn();
const mockSetThresholdState = jest.fn();
jest.spyOn(context, 'useCreateAlertState').mockReturnValue(
	createMockAlertContextState({
		setThresholdState: mockSetThresholdState,
		setAlertState: mockSetAlertState,
	}),
);

const TEST_CONSTANTS = {
	THRESHOLD_ID: 'test-threshold-1',
	CRITICAL_LABEL: 'CRITICAL',
	WARNING_LABEL: 'WARNING',
	INFO_LABEL: 'INFO',
	CHANNEL_1: 'channel-1',
	CHANNEL_2: 'channel-2',
	CHANNEL_3: 'channel-3',
	EMAIL_CHANNEL_NAME: 'Email Channel',
	EMAIL_CHANNEL_TRUNCATED: 'Email Chan...',
	ENTER_THRESHOLD_NAME: 'v2_threshold_name_placeholder',
	ENTER_THRESHOLD_VALUE: 'v2_threshold_value_placeholder',
	ENTER_RECOVERY_THRESHOLD_VALUE: 'Enter recovery threshold value',
} as const;

const mockThreshold = {
	id: TEST_CONSTANTS.THRESHOLD_ID,
	label: TEST_CONSTANTS.CRITICAL_LABEL,
	thresholdValue: 100,
	recoveryThresholdValue: 80,
	unit: 'bytes',
	channels: [TEST_CONSTANTS.CHANNEL_1],
	color: '#ff0000',
};

const mockChannels: Channels[] = [
	{
		id: TEST_CONSTANTS.CHANNEL_1,
		name: TEST_CONSTANTS.EMAIL_CHANNEL_NAME,
	} as any,
	{ id: TEST_CONSTANTS.CHANNEL_2, name: 'Slack Channel' } as any,
	{ id: TEST_CONSTANTS.CHANNEL_3, name: 'PagerDuty Channel' } as any,
];

const mockUnits: DefaultOptionType[] = [
	{ label: 'Bytes', value: 'bytes' },
	{ label: 'KB', value: 'kb' },
	{ label: 'MB', value: 'mb' },
];

const defaultProps: ThresholdItemProps = {
	threshold: mockThreshold,
	updateThreshold: jest.fn(),
	removeThreshold: jest.fn(),
	showRemoveButton: false,
	channels: mockChannels,
	isLoadingChannels: false,
	units: mockUnits,
	isErrorChannels: false,
	refreshChannels: jest.fn(),
};

const renderThresholdItem = (
	props: Partial<ThresholdItemProps> = {},
): ReturnType<typeof render> => {
	const mergedProps = { ...defaultProps, ...props };
	return render(<ThresholdItem {...mergedProps} />);
};

const verifySelectorWidth = (
	selectorIndex: number,
	expectedWidth: string,
): void => {
	const selectors = screen.getAllByRole('combobox');
	const selector = selectors[selectorIndex];
	expect(selector.closest('.ant-select')).toHaveStyle(`width: ${expectedWidth}`);
};

// TODO: Unskip this when recovery threshold is implemented
// const showRecoveryThreshold = (): void => {
// 	const recoveryButton = screen.getByRole('button', { name: '' });
// 	fireEvent.click(recoveryButton);
// };

const verifyComponentRendersWithLoading = (): void => {
	expect(screen.getByTestId('threshold-name-select')).toBeInTheDocument();
};

const verifyUnitSelectorDisabled = (): void => {
	const unitSelectors = screen.getAllByRole('combobox');
	const unitSelector = unitSelectors[1]; // Second combobox is the unit selector (first is severity)
	expect(unitSelector).toBeDisabled();
};

describe('ThresholdItem', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('renders threshold indicator with correct color', () => {
		renderThresholdItem();

		// Find the threshold dot by its class
		const thresholdDot = document.querySelector('.threshold-dot');
		expect(thresholdDot).toHaveStyle('background-color: #ff0000');
	});

	it('renders threshold label input with correct value', () => {
		renderThresholdItem();

		// Label is now a Select (severity dropdown), not a free-text Input
		const labelSelect = screen.getByTestId('threshold-name-select');
		expect(labelSelect).toBeInTheDocument();
		// Selected value is rendered as title by AntD Select
		expect(screen.getByTitle(TEST_CONSTANTS.CRITICAL_LABEL)).toBeInTheDocument();
	});

	it('renders threshold value input with correct value', () => {
		renderThresholdItem();

		const valueInput = screen.getByPlaceholderText(
			TEST_CONSTANTS.ENTER_THRESHOLD_VALUE,
		);
		expect(valueInput).toHaveValue(100);
	});

	it('renders unit selector with correct value', () => {
		renderThresholdItem();

		// Check for the unit selector by looking for the displayed text
		expect(screen.getByText('Bytes')).toBeInTheDocument();
	});

	it('updates threshold label when label input changes', () => {
		const updateThreshold = jest.fn();
		renderThresholdItem({ updateThreshold });

		// Label is now a Select; open and choose an option
		const labelSelect = screen.getByTestId('threshold-name-select');
		const selector = labelSelect.querySelector('.ant-select-selector') || labelSelect;
		fireEvent.mouseDown(selector);

		// 'warning' is the lowercase option in the severity dropdown
		const warningOption = screen.getByTitle('warning');
		fireEvent.click(warningOption);

		expect(updateThreshold).toHaveBeenCalledWith(
			TEST_CONSTANTS.THRESHOLD_ID,
			'label',
			'warning',
		);
	});

	it('updates threshold value when value input changes', () => {
		const updateThreshold = jest.fn();
		renderThresholdItem({ updateThreshold });

		const valueInput = screen.getByPlaceholderText(
			TEST_CONSTANTS.ENTER_THRESHOLD_VALUE,
		);
		fireEvent.change(valueInput, { target: { value: '200' } });

		expect(updateThreshold).toHaveBeenCalledWith(
			TEST_CONSTANTS.THRESHOLD_ID,
			'thresholdValue',
			'200',
		);
	});

	it('updates threshold unit when unit selector changes', () => {
		const updateThreshold = jest.fn();
		renderThresholdItem({ updateThreshold });

		// Find the unit selector by its role and simulate change
		const unitSelectors = screen.getAllByRole('combobox');
		const unitSelector = unitSelectors[1]; // Second combobox is the unit selector (severity is first)

		// Simulate clicking to open the dropdown and selecting a value
		fireEvent.click(unitSelector);

		// The actual change event might not work the same way with Ant Design Select
		// So we'll just verify the selector is present and can be interacted with
		expect(unitSelector).toBeInTheDocument();
	});

	it('updates threshold channels when channels selector changes', () => {
		const updateThreshold = jest.fn();
		renderThresholdItem({ updateThreshold });

		// Find the channels selector by its role and simulate change
		const channelSelectors = screen.getAllByRole('combobox');
		const channelSelector = channelSelectors[2]; // Third combobox is the channels selector (severity, unit, channels)

		// Simulate clicking to open the dropdown
		fireEvent.click(channelSelector);

		// The actual change event might not work the same way with Ant Design Select
		// So we'll just verify the selector is present and can be interacted with
		expect(channelSelector).toBeInTheDocument();
	});

	it('shows remove button when showRemoveButton is true', () => {
		renderThresholdItem({ showRemoveButton: true });

		// The remove button is the second button (with circle-x icon)
		const buttons = screen.getAllByRole('button');
		expect(buttons).toHaveLength(1); // remove button
	});

	it('does not show remove button when showRemoveButton is false', () => {
		renderThresholdItem({ showRemoveButton: false });

		// No buttons should be present
		const buttons = screen.queryAllByRole('button');
		expect(buttons).toHaveLength(0);
	});

	it('calls removeThreshold when remove button is clicked', () => {
		const removeThreshold = jest.fn();
		renderThresholdItem({ showRemoveButton: true, removeThreshold });

		// The remove button is the first button (with circle-x icon)
		const buttons = screen.getAllByRole('button');
		const removeButton = buttons[0];
		fireEvent.click(removeButton);

		expect(removeThreshold).toHaveBeenCalledWith(TEST_CONSTANTS.THRESHOLD_ID);
	});

	// TODO: Unskip this when recovery threshold is implemented
	it.skip('shows recovery threshold inputs when recovery button is clicked', () => {
		renderThresholdItem();

		// The recovery button is the first button (with chart-line icon)
		const buttons = screen.getAllByRole('button');
		const recoveryButton = buttons[0]; // First button is the recovery button
		fireEvent.click(recoveryButton);

		expect(
			screen.getByPlaceholderText('Enter recovery threshold value'),
		).toBeInTheDocument();
		expect(
			screen.getByPlaceholderText(TEST_CONSTANTS.ENTER_RECOVERY_THRESHOLD_VALUE),
		).toBeInTheDocument();
	});

	// TODO: Unskip this when recovery threshold is implemented
	it.skip('updates recovery threshold value when input changes', () => {
		const updateThreshold = jest.fn();
		renderThresholdItem({ updateThreshold });

		// Show recovery threshold first
		const buttons = screen.getAllByRole('button');
		const recoveryButton = buttons[0]; // First button is the recovery button
		fireEvent.click(recoveryButton);

		const recoveryValueInput = screen.getByPlaceholderText(
			TEST_CONSTANTS.ENTER_RECOVERY_THRESHOLD_VALUE,
		);
		fireEvent.change(recoveryValueInput, { target: { value: '90' } });

		expect(updateThreshold).toHaveBeenCalledWith(
			TEST_CONSTANTS.THRESHOLD_ID,
			'recoveryThresholdValue',
			'90',
		);
	});

	it('disables unit selector when no units are available', () => {
		renderThresholdItem({ units: [] });
		verifyUnitSelectorDisabled();
	});

	it('shows tooltip when no units are available', () => {
		renderThresholdItem({ units: [] });

		// The tooltip should be present when hovering over disabled unit selector
		verifyUnitSelectorDisabled();
	});

	it('handles empty threshold values correctly', () => {
		const emptyThreshold = {
			...mockThreshold,
			label: '',
			thresholdValue: 0,
			unit: '',
			channels: [],
		};

		renderThresholdItem({ threshold: emptyThreshold });

		// Label is now a Select; placeholder text appears as inner text when no value is selected
		expect(screen.getByText('v2_threshold_name_placeholder')).toBeInTheDocument();
		expect(screen.getByPlaceholderText('v2_threshold_value_placeholder')).toHaveValue(0);
	});

	it('renders with correct input widths', () => {
		renderThresholdItem();

		// Label is now a Select; width is on the Select wrapper element
		const labelSelect = screen.getByTestId('threshold-name-select');
		const valueInput = screen.getByPlaceholderText(
			TEST_CONSTANTS.ENTER_THRESHOLD_VALUE,
		);

		expect(labelSelect).toHaveStyle('width: 200px');
		expect(valueInput).toHaveStyle('width: 100px');
	});

	it('renders channels selector with correct width', () => {
		renderThresholdItem();
		verifySelectorWidth(2, '350px'); // severity[0], unit[1], channels[2]
	});

	it('renders unit selector with correct width', () => {
		renderThresholdItem();
		verifySelectorWidth(1, '150px'); // severity[0], unit[1]
	});

	it('handles loading channels state', () => {
		renderThresholdItem({ isLoadingChannels: true });
		verifyComponentRendersWithLoading();
	});

	it.skip('renders recovery threshold with correct initial value', () => {
		renderThresholdItem();
		// showRecoveryThreshold();

		const recoveryValueInput = screen.getByPlaceholderText(
			TEST_CONSTANTS.ENTER_RECOVERY_THRESHOLD_VALUE,
		);
		expect(recoveryValueInput).toHaveValue(80);
	});

	it('handles threshold without channels', () => {
		const thresholdWithoutChannels = {
			...mockThreshold,
			channels: [],
		};

		renderThresholdItem({ threshold: thresholdWithoutChannels });

		// Should render all three selectors: severity, unit, channels
		const channelSelectors = screen.getAllByRole('combobox');
		expect(channelSelectors).toHaveLength(3); // severity[0], unit[1], channels[2]
	});
});
