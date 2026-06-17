import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { QueryBuilderProvider } from 'providers/QueryBuilder';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import * as rulesHook from '../../../api/generated/services/rules';
import { CreateAlertProvider, useCreateAlertState } from '../context';
import { getMissingOperationalLabels } from '../CreateAlertHeader/operationalMetadata';

// The CreateAlertProvider instantiates these mutation hooks at mount.
jest.spyOn(rulesHook, 'useCreateRule').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useCreateRule>);
jest.spyOn(rulesHook, 'useTestRule').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useTestRule>);
jest.spyOn(rulesHook, 'useUpdateRuleByID').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useUpdateRuleByID>);

// Drives the REAL query builder state and reads the REAL alert labels so we can
// assert the query -> label auto-sync end to end through the actual providers.
function Harness(): JSX.Element {
	const { currentQuery, handleSetQueryData } = useQueryBuilder();
	const { alertState } = useCreateAlertState();

	const setFilter = (expression: string): void => {
		const firstQuery = currentQuery.builder.queryData[0];
		handleSetQueryData(0, {
			...firstQuery,
			filter: { ...firstQuery.filter, expression },
		});
	};

	return (
		<div>
			<button
				type="button"
				data-testid="set-service"
				onClick={(): void => setFilter(`service.name = 'frontend'`)}
			>
				set service
			</button>
			<button
				type="button"
				data-testid="set-service-and-env"
				onClick={(): void =>
					setFilter(
						`service.name = 'frontend' AND deployment.environment = 'prod'`,
					)
				}
			>
				set service + env
			</button>
			<button
				type="button"
				data-testid="clear-filter"
				onClick={(): void => setFilter('')}
			>
				clear
			</button>
			<div data-testid="labels">{JSON.stringify(alertState.labels)}</div>
			<div data-testid="missing-operational">
				{getMissingOperationalLabels(alertState.labels)
					.map((l) => l.key)
					.join(',')}
			</div>
		</div>
	);
}

const renderHarness = (): ReturnType<typeof render> =>
	render(
		<MemoryRouter>
			<QueryBuilderProvider>
				<CreateAlertProvider initialAlertType={AlertTypes.METRICS_BASED_ALERT}>
					<Harness />
				</CreateAlertProvider>
			</QueryBuilderProvider>
		</MemoryRouter>,
	);

describe('CreateAlertV2 query <-> label auto-sync (real providers)', () => {
	it('adds service_name label when service.name filter is set in the query', async () => {
		renderHarness();

		expect(screen.getByTestId('labels')).toHaveTextContent('{}');

		fireEvent.click(screen.getByTestId('set-service'));

		await waitFor(() => {
			expect(screen.getByTestId('labels')).toHaveTextContent(
				'"service.name":"frontend"',
			);
		});
	});

	it('syncs multiple managed attributes', async () => {
		renderHarness();

		fireEvent.click(screen.getByTestId('set-service-and-env'));

		await waitFor(() => {
			const text = screen.getByTestId('labels').textContent || '';
			expect(text).toContain('"service.name":"frontend"');
			expect(text).toContain('"deployment_environment":"prod"');
		});
	});

	it('marks the recommended service.name operational label as present once synced', async () => {
		renderHarness();

		// Initially the recommended "Service Name" label is missing.
		expect(screen.getByTestId('missing-operational')).toHaveTextContent(
			'service.name',
		);

		fireEvent.click(screen.getByTestId('set-service'));

		// After the query filter sets service.name, it is no longer missing.
		await waitFor(() => {
			expect(screen.getByTestId('missing-operational')).not.toHaveTextContent(
				'service.name',
			);
		});
	});

	it('removes the managed label when the filter is cleared', async () => {
		renderHarness();

		fireEvent.click(screen.getByTestId('set-service'));
		await waitFor(() => {
			expect(screen.getByTestId('labels')).toHaveTextContent('service.name');
		});

		fireEvent.click(screen.getByTestId('clear-filter'));
		await waitFor(() => {
			expect(screen.getByTestId('labels')).not.toHaveTextContent('service.name');
		});
	});
});
