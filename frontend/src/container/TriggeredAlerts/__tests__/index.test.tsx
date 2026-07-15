import { render, screen } from '@testing-library/react';
import { useQuery } from 'react-query';

import TriggeredAlerts from '../index';

jest.mock('react-query', () => ({
	useQuery: jest.fn(),
}));

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));

jest.mock('providers/App/App', () => ({
	useAppContext: () => ({ user: { id: 'user-1' } }),
}));

jest.mock('hooks/useAxiosError', () => ({
	__esModule: true,
	default: () => jest.fn(),
}));

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: jest.fn(),
}));

jest.mock('api/alerts/getTriggered', () => ({
	__esModule: true,
	default: jest.fn(),
}));

jest.mock('../TriggeredAlert', () => ({
	__esModule: true,
	default: function MockTriggerComponent(): JSX.Element {
		return <div data-testid="trigger-component" />;
	},
}));

const useQueryMock = useQuery as jest.Mock;

describe('TriggeredAlerts', () => {
	it('keeps the table mounted during a background refetch', () => {
		useQueryMock.mockReturnValue({
			isLoading: false,
			isFetching: true,
			error: undefined,
			data: { payload: [] },
		});

		render(<TriggeredAlerts />);

		expect(screen.getByTestId('trigger-component')).toBeInTheDocument();
	});

	it('shows the spinner only on initial load', () => {
		useQueryMock.mockReturnValue({
			isLoading: true,
			isFetching: true,
			error: undefined,
			data: undefined,
		});

		render(<TriggeredAlerts />);

		expect(
			screen.queryByTestId('trigger-component'),
		).not.toBeInTheDocument();
	});
});
