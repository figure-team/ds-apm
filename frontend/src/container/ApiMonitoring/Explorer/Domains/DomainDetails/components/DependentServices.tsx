import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { UseQueryResult } from 'react-query';
import { Skeleton, Table, TablePaginationConfig, Typography } from 'antd';
import { QueryParams } from 'constants/query';
import {
	DependentServicesData,
	getDependentServicesColumns,
	getFormattedDependentServicesData,
} from 'container/ApiMonitoring/utils';
import { UnfoldVertical } from 'lucide-react';
import { SuccessResponse } from 'types/api';
import { openInNewTab } from 'utils/navigation';

import emptyStateUrl from '@/assets/Icons/emptyState.svg';

import ErrorState from './ErrorState';

import '../DomainDetails.styles.scss';

interface DependentServicesProps {
	dependentServicesQuery: UseQueryResult<SuccessResponse<any>, unknown>;
	timeRange: {
		startTime: number;
		endTime: number;
	};
}

function DependentServices({
	dependentServicesQuery,
	timeRange,
}: DependentServicesProps): JSX.Element {
	const { data, refetch, isError, isLoading, isRefetching } =
		dependentServicesQuery;
	const { t } = useTranslation('apiMonitoring');

	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	const handleShowMoreClick = (): void => {
		setIsExpanded((prev) => !prev);
	};

	const dependentServicesData = useMemo(
		(): DependentServicesData[] =>
			getFormattedDependentServicesData(data?.payload?.data?.result[0].table.rows),
		[data],
	);

	const paginationConfig = useMemo(
		(): TablePaginationConfig => ({
			pageSize: isExpanded ? dependentServicesData.length : 5,
			hideOnSinglePage: true,
			position: ['none', 'none'],
		}),
		[isExpanded, dependentServicesData.length],
	);

	if (isLoading || isRefetching) {
		return <Skeleton />;
	}

	if (isError) {
		return <ErrorState refetch={refetch} />;
	}

	return (
		<div className="top-services-content">
			<div className="dependent-services-container">
				<Table
					loading={isLoading || isRefetching}
					dataSource={dependentServicesData || []}
					columns={getDependentServicesColumns(t)}
					rowClassName="table-row-dark"
					pagination={paginationConfig}
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
					onRow={(record): { onClick: () => void; className: string } => ({
						onClick: (): void => {
							const serviceName =
								record.serviceData.serviceName && record.serviceData.serviceName !== '-'
									? record.serviceData.serviceName
									: '';
							const urlQuery = new URLSearchParams();
							urlQuery.set(QueryParams.startTime, timeRange.startTime.toString());
							urlQuery.set(QueryParams.endTime, timeRange.endTime.toString());
							openInNewTab(`/services/${serviceName}?${urlQuery.toString()}`);
						},
						className: 'clickable-row',
					})}
				/>

				{dependentServicesData.length > 5 && (
					<div
						className="top-services-load-more"
						onClick={handleShowMoreClick}
						onKeyDown={(e): void => {
							if (e.key === 'Enter') {
								handleShowMoreClick();
							}
						}}
						role="button"
						tabIndex={0}
					>
						<UnfoldVertical size={14} />
						{isExpanded ? t('show_less') : t('show_more')}
					</div>
				)}
			</div>
		</div>
	);
}

export default DependentServices;
