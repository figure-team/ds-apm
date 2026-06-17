import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Table, TableProps, Typography } from 'antd';
import { RotateCw } from 'lucide-react';

import awwSnapUrl from '@/assets/Icons/awwSnap.svg';
import emptyStateUrl from '@/assets/Icons/emptyState.svg';

import RoutingPolicyListItem from './RoutingPolicyListItem';
import { RoutingPolicy, RoutingPolicyListProps } from './types';

function RoutingPolicyList({
	routingPolicies,
	refetchRoutingPolicies,
	isRoutingPoliciesFetching,
	isRoutingPoliciesLoading,
	isRoutingPoliciesError,
	handlePolicyDetailsModalOpen,
	handleDeleteModalOpen,
	hasSearchTerm,
}: RoutingPolicyListProps): JSX.Element {
	const { t } = useTranslation('alerts');
	const columns: TableProps<RoutingPolicy>['columns'] = [
		{
			title: t('rp_col'),
			key: 'routingPolicy',
			render: (data: RoutingPolicy): JSX.Element => (
				<RoutingPolicyListItem
					routingPolicy={data}
					handlePolicyDetailsModalOpen={handlePolicyDetailsModalOpen}
					handleDeleteModalOpen={handleDeleteModalOpen}
				/>
			),
		},
	];

	const showLoading = isRoutingPoliciesLoading || isRoutingPoliciesFetching;
	const showError = !showLoading && isRoutingPoliciesError;

	const localeEmptyState = useMemo(
		() => (
			<div className="no-routing-policies-message-container">
				{showError ? (
					<img src={awwSnapUrl} alt="aww-snap" className="error-state-svg" />
				) : (
					<img
						src={emptyStateUrl}
						alt="thinking-emoji"
						className="empty-state-svg"
					/>
				)}
				{showError ? (
					<div className="error-state">
						<Typography.Text>{t('rp_fetch_error')}</Typography.Text>
						<Button icon={<RotateCw size={14} />} onClick={refetchRoutingPolicies}>
							{t('rp_retry')}
						</Button>
					</div>
				) : hasSearchTerm ? (
					<Typography.Text>{t('rp_no_match')}</Typography.Text>
				) : (
					<Typography.Text>
						{t('rp_empty_prefix')}{' '}
						<a
							href="https://signoz.io/docs/alerts-management/routing-policy"
							target="_blank"
							rel="noopener noreferrer"
						>
							{t('rp_learn_more')}
						</a>
					</Typography.Text>
				)}
			</div>
		),
		[showError, hasSearchTerm, refetchRoutingPolicies, t],
	);

	return (
		<Table<RoutingPolicy>
			columns={columns}
			className="routing-policies-table"
			bordered={false}
			dataSource={routingPolicies}
			loading={showLoading}
			showHeader={false}
			rowKey="id"
			pagination={{
				pageSize: 5,
				showSizeChanger: false,
				hideOnSinglePage: true,
			}}
			locale={{
				emptyText: showLoading ? null : localeEmptyState,
			}}
		/>
	);
}

export default RoutingPolicyList;
