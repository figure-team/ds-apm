import { Typography } from 'antd';
import getDLQEntries from 'api/dlq/getDLQEntries';
import replayDLQEntries from 'api/dlq/replayDLQEntries';
import Spinner from 'components/Spinner';
import { useNotifications } from 'hooks/useNotifications';
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from 'react-query';
import { DLQEntry, GetDLQEntriesParams } from 'types/api/dlq';

import { DLQBulkActionBar } from './DLQBulkActionBar';
import { DLQFilters } from './DLQFilters';
import { DLQPayloadDrawer } from './DLQPayloadDrawer';
import { DLQTable } from './DLQTable';

const DLQ_ENTRIES_KEY = 'dlqEntries';

function DLQFailures(): JSX.Element {
	const { notifications } = useNotifications();
	const queryClient = useQueryClient();

	const [filters, setFilters] = useState<GetDLQEntriesParams>({});
	const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
	const [drawerPayload, setDrawerPayload] = useState<string | null>(null);

	const { data, isLoading, error } = useQuery<DLQEntry[], Error>(
		[DLQ_ENTRIES_KEY, filters],
		async () => {
			const res = await getDLQEntries(filters);
			return res.data ?? [];
		},
	);

	const replayMutation = useMutation(
		(eventIDs: string[]) =>
			replayDLQEntries({ event_ids: eventIDs }).then((r) => r.data),
		{
			onSuccess: (result) => {
				notifications.success({
					message: `재전송 완료: ${result.replayed}건 성공, ${result.skipped}건 skip, ${result.failed}건 실패`,
				});
				setSelectedKeys([]);
				void queryClient.invalidateQueries([DLQ_ENTRIES_KEY]);
			},
			onError: () => {
				notifications.error({ message: '재전송 요청이 실패했습니다.' });
			},
		},
	);

	if (isLoading) return <Spinner tip="전송 실패 내역 로딩 중..." height="60vh" />;

	if (error) {
		return <Typography.Text type="danger">{error.message}</Typography.Text>;
	}

	const entries = data ?? [];

	return (
		<div>
			<DLQFilters
				entries={entries}
				filters={filters}
				onChange={(f): void => {
					setFilters(f);
					setSelectedKeys([]);
				}}
			/>
			<DLQBulkActionBar
				selectedCount={selectedKeys.length}
				loading={replayMutation.isLoading}
				onReplay={(): void => {
					replayMutation.mutate(selectedKeys);
				}}
				onClear={(): void => setSelectedKeys([])}
			/>
			<DLQTable
				entries={entries}
				selectedKeys={selectedKeys}
				onSelectionChange={setSelectedKeys}
				onViewPayload={(payload): void => setDrawerPayload(payload)}
			/>
			<DLQPayloadDrawer
				payload={drawerPayload}
				onClose={(): void => setDrawerPayload(null)}
			/>
		</div>
	);
}

export default DLQFailures;
