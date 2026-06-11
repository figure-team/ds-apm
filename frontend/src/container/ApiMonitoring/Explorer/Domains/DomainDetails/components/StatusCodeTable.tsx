import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { UseQueryResult } from 'react-query';
import { Table, Typography } from 'antd';
import {
	getEndPointStatusCodeColumns,
	getFormattedEndPointStatusCodeData,
} from 'container/ApiMonitoring/utils';
import { SuccessResponse } from 'types/api';

import emptyStateUrl from '@/assets/Icons/emptyState.svg';

import ErrorState from './ErrorState';

function StatusCodeTable({
	endPointStatusCodeDataQuery,
}: {
	endPointStatusCodeDataQuery: UseQueryResult<SuccessResponse<any>, unknown>;
}): JSX.Element {
	const { isLoading, isRefetching, isError, data, refetch } =
		endPointStatusCodeDataQuery;
	const { t } = useTranslation('apiMonitoring');

	const statusCodeData = useMemo(() => {
		if (isLoading || isRefetching || isError) {
			return [];
		}

		return getFormattedEndPointStatusCodeData(
			data?.payload?.data?.result[0].table.rows,
		);
	}, [data?.payload?.data?.result, isLoading, isRefetching, isError]);

	if (isError) {
		return <ErrorState refetch={refetch} />;
	}

	return (
		<div className="status-code-table-container">
			<Table
				loading={isLoading || isRefetching}
				dataSource={statusCodeData || []}
				columns={getEndPointStatusCodeColumns(t)}
				pagination={false}
				rowClassName={(_, index): string =>
					index % 2 === 0 ? 'table-row-dark' : 'table-row-light'
				}
				locale={{
					emptyText:
						isLoading || isRefetching ? null : (
							<div className="no-status-code-data-message-container">
								<div className="no-status-code-data-message-content">
									<img
										src={emptyStateUrl}
										alt="thinking-emoji"
										className="empty-state-svg"
									/>

									<Typography.Text className="no-status-code-data-message">
										{t('query_no_results')}
									</Typography.Text>
								</div>
							</div>
						),
				}}
			/>
		</div>
	);
}

export default StatusCodeTable;
