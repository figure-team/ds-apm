import {
	allRoles,
	listRolesSuccessResponse,
} from 'mocks-server/__mockdata__/roles';
import { server } from 'mocks-server/server';
import { rest } from 'msw';
import { render, screen, userEvent } from 'tests/test-utils';

import RolesSettings from '../RolesSettings';

const rolesApiURL = 'http://localhost/api/v1/roles';

describe('RolesSettings', () => {
	afterEach(() => {
		jest.clearAllMocks();
	});

	it('renders the header and search input', () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(screen.getByText('roles_title')).toBeInTheDocument();
		expect(screen.getByText('roles_subtitle')).toBeInTheDocument();
		expect(
			screen.getByPlaceholderText('search_roles_placeholder'),
		).toBeInTheDocument();
	});

	it('displays roles grouped by managed and custom sections', async () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('signoz-admin')).toBeInTheDocument();

		// Section headers
		expect(screen.getByText('managed_roles')).toBeInTheDocument();
		expect(screen.getByText('custom_roles')).toBeInTheDocument();

		// Managed roles
		expect(screen.getByText('signoz-admin')).toBeInTheDocument();
		expect(screen.getByText('signoz-editor')).toBeInTheDocument();
		expect(screen.getByText('signoz-viewer')).toBeInTheDocument();

		// Custom roles
		expect(screen.getByText('billing-manager')).toBeInTheDocument();
		expect(screen.getByText('dashboard-creator')).toBeInTheDocument();

		// Custom roles count badge
		expect(screen.getByText('2')).toBeInTheDocument();

		// Column headers
		expect(screen.getByText('name')).toBeInTheDocument();
		expect(screen.getByText('description')).toBeInTheDocument();
		expect(screen.getByText('updated_at')).toBeInTheDocument();
		expect(screen.getByText('created_at')).toBeInTheDocument();
	});

	it('filters roles by search query on name', async () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('signoz-admin')).toBeInTheDocument();

		const user = userEvent.setup({ pointerEventsCheck: 0 });
		const searchInput = screen.getByPlaceholderText('search_roles_placeholder');

		await user.type(searchInput, 'billing');

		expect(await screen.findByText('billing-manager')).toBeInTheDocument();
		expect(screen.queryByText('signoz-admin')).not.toBeInTheDocument();
		expect(screen.queryByText('signoz-editor')).not.toBeInTheDocument();
		expect(screen.queryByText('dashboard-creator')).not.toBeInTheDocument();
	});

	it('filters roles by search query on description', async () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('signoz-admin')).toBeInTheDocument();

		const user = userEvent.setup({ pointerEventsCheck: 0 });
		const searchInput = screen.getByPlaceholderText('search_roles_placeholder');

		await user.type(searchInput, 'read-only');

		expect(await screen.findByText('signoz-viewer')).toBeInTheDocument();
		expect(screen.queryByText('signoz-admin')).not.toBeInTheDocument();
		expect(screen.queryByText('billing-manager')).not.toBeInTheDocument();
	});

	it('shows empty state when search matches nothing', async () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('signoz-admin')).toBeInTheDocument();

		const user = userEvent.setup({ pointerEventsCheck: 0 });
		const searchInput = screen.getByPlaceholderText('search_roles_placeholder');

		await user.type(searchInput, 'nonexistentrole');

		expect(
			await screen.findByText('no_roles_match'),
		).toBeInTheDocument();
	});

	it('shows loading skeleton while fetching', () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.delay(200), ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(document.querySelector('.ant-skeleton')).toBeInTheDocument();
	});

	it('shows error state when API fails', async () => {
		const errorMessage = 'Failed to fetch roles';
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(
					ctx.status(500),
					ctx.json({
						error: {
							code: 'INTERNAL_ERROR',
							message: errorMessage,
							url: '',
							errors: [],
						},
					}),
				),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText(errorMessage)).toBeInTheDocument();
	});

	it('shows empty state when API returns no roles', async () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json({ status: 'success', data: [] })),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('no_roles_found')).toBeInTheDocument();
	});

	it('renders descriptions for all roles', async () => {
		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(ctx.status(200), ctx.json(listRolesSuccessResponse)),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('signoz-admin')).toBeInTheDocument();

		for (const role of allRoles) {
			if (role.description) {
				expect(screen.getByText(role.description)).toBeInTheDocument();
			}
		}
	});

	it('handles invalid dates gracefully by showing fallback', async () => {
		const invalidRole = {
			id: 'edge-0009',
			createdAt: 'invalid-date' as unknown as Date,
			updatedAt: 'not-a-date' as unknown as Date,
			name: 'invalid-date-role',
			description: 'Tests date parsing fallback.',
			type: 'custom',
			orgId: 'org-001',
		};

		server.use(
			rest.get(rolesApiURL, (_req, res, ctx) =>
				res(
					ctx.status(200),
					ctx.json({
						status: 'success',
						data: [invalidRole],
					}),
				),
			),
		);

		render(<RolesSettings />);

		expect(await screen.findByText('invalid-date-role')).toBeInTheDocument();

		// Verify the "—" (em-dash) fallback is shown for both cells
		const dashFallback = screen.getAllByText('—');
		// In renderRow: name, description, updatedAt, createdAt.
		// Total dashes expected: 2 (for both dates)
		expect(dashFallback.length).toBeGreaterThanOrEqual(2);
	});
});
