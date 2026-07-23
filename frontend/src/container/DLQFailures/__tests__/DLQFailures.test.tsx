import DLQFailures from 'container/DLQFailures';
import { act, fireEvent, render, screen, waitFor } from 'tests/test-utils';

jest.mock('api/dlq/getDLQEntries', () => ({
	__esModule: true,
	default: jest.fn(() =>
		Promise.resolve({
			httpStatusCode: 200,
			data: [
				{
					event_id: 'abc123def456gh78',
					channel: 'slack',
					failed_at: '2026-06-15T09:23:11Z',
					reason: 'connection refused',
					status: 'pending',
					payload: btoa(JSON.stringify([{ labels: { alertname: 'test' } }])),
				},
				{
					event_id: 'xyz789abc123de45',
					channel: 'webhook',
					failed_at: '2026-06-15T10:00:00Z',
					reason: '5xx',
					status: 'replayed',
					payload: btoa(JSON.stringify([])),
				},
			],
		}),
	),
}));

jest.mock('api/dlq/replayDLQEntries', () => ({
	__esModule: true,
	default: jest.fn(() =>
		Promise.resolve({
			httpStatusCode: 200,
			data: { replayed: 1, skipped: 0, failed: 0 },
		}),
	),
}));

jest.mock('api/channels/getAll', () => ({
	__esModule: true,
	default: jest.fn(() => Promise.resolve({ httpStatusCode: 200, data: [] })),
}));

jest.mock('hooks/useNotifications', () => ({
	__esModule: true,
	useNotifications: jest.fn(() => ({
		notifications: { success: jest.fn(), error: jest.fn() },
	})),
}));

describe('DLQFailures', () => {
	it('테이블에 DLQ 항목이 렌더링된다', async () => {
		render(<DLQFailures />);
		await waitFor(() => {
			expect(screen.getByText('abc123def456')).toBeInTheDocument();
		});
		expect(screen.getByText('xyz789abc123')).toBeInTheDocument();
	});

	it('체크박스 선택 시 BulkActionBar가 나타난다', async () => {
		render(<DLQFailures />);
		await waitFor(() => {
			expect(screen.getByText('abc123def456')).toBeInTheDocument();
		});

		const checkboxes = screen.getAllByRole('checkbox');
		await act(async () => {
			fireEvent.click(checkboxes[1]);
		});

		expect(screen.getByText('dlq_selected_count')).toBeInTheDocument();
		expect(screen.getByText('dlq_btn_replay')).toBeInTheDocument();
	});

	it('재전송 버튼 클릭 시 replayDLQEntries를 호출한다', async () => {
		const { default: replayDLQEntries } = await import('api/dlq/replayDLQEntries');
		render(<DLQFailures />);
		await waitFor(() => {
			expect(screen.getByText('abc123def456')).toBeInTheDocument();
		});

		const checkboxes = screen.getAllByRole('checkbox');
		await act(async () => {
			fireEvent.click(checkboxes[1]);
		});

		await act(async () => {
			fireEvent.click(screen.getByText('dlq_btn_replay'));
		});

		await waitFor(() => {
			expect(replayDLQEntries).toHaveBeenCalledWith({
				event_ids: ['abc123def456gh78'],
			});
		});
	});

	it('"보기" 버튼 클릭 시 Drawer가 열린다', async () => {
		render(<DLQFailures />);
		await waitFor(() => {
			expect(screen.getByText('abc123def456')).toBeInTheDocument();
		});

		const viewButtons = screen.getAllByText('dlq_btn_view');
		await act(async () => {
			fireEvent.click(viewButtons[0]);
		});

		await waitFor(() => {
			expect(screen.getByText('dlq_drawer_title')).toBeInTheDocument();
		});
	});
});

describe('AllAlertChannels with Tabs', () => {
	it('두 탭이 렌더링된다', async () => {
		const { default: AlertChannels } = await import('container/AllAlertChannels');
		render(<AlertChannels />);

		await waitFor(() => {
			expect(screen.getByText('tab_channel_list')).toBeInTheDocument();
			expect(screen.getByText('tab_dlq')).toBeInTheDocument();
		});
	});
});
