import { TableColumnType as ColumnType, Tooltip } from 'antd';
import { TFunction } from 'i18next';
import { Group } from 'lucide-react';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';

import { K8sRenderedRowData } from '../Base/types';
import { IEntityColumn } from '../Base/useInfraMonitoringTableColumnsStore';
import { getGroupByEl, getGroupedByMeta, getRowKey } from '../Base/utils';
import {
	EntityProgressBar,
	formatBytes,
	ValidateColumnValueWrapper,
} from '../commonUtils';
import { InfraMonitoringEntity } from '../constants';
import { K8sJobsData } from './api';

import styles from './table.module.scss';

export const k8sJobsColumns: IEntityColumn[] = [
	{
		label: 'col_job_group',
		value: 'jobGroup',
		id: 'jobGroup',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-collapse',
	},
	{
		label: 'col_job_name',
		value: 'jobName',
		id: 'jobName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-expand',
	},
	{
		label: 'col_namespace_name',
		value: 'namespaceName',
		id: 'namespaceName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_successful',
		value: 'successful_pods',
		id: 'successful_pods',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_failed',
		value: 'failed_pods',
		id: 'failed_pods',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_desired_successful',
		value: 'desired_successful_pods',
		id: 'desired_successful_pods',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_active',
		value: 'active_pods',
		id: 'active_pods',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_cpu_req_usage_pct',
		value: 'cpu_request',
		id: 'cpu_request',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_cpu_limit_usage_pct',
		value: 'cpu_limit',
		id: 'cpu_limit',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_cpu_usage_cores',
		value: 'cpu',
		id: 'cpu',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_mem_req_usage_pct',
		value: 'memory_request',
		id: 'memory_request',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_mem_limit_usage_pct',
		value: 'memory_limit',
		id: 'memory_limit',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_mem_usage_wss',
		value: 'memory',
		id: 'memory',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
];

export const k8sJobsColumnsConfig = (
	t: TFunction,
): ColumnType<K8sRenderedRowData>[] => [
	{
		title: (
			<div className={styles.entityGroupHeader}>
				<Group size={14} /> {t('col_job_group').toString()}
			</div>
		),
		dataIndex: 'jobGroup',
		key: 'jobGroup',
		ellipsis: true,
		width: 150,
		align: 'left',
		sorter: false,
	},
	{
		title: <div>{t('col_job_name').toString()}</div>,
		dataIndex: 'jobName',
		key: 'jobName',
		ellipsis: true,
		width: 250,
		sorter: false,
		align: 'left',
	},
	{
		title: <div>{t('col_namespace_name').toString()}</div>,
		dataIndex: 'namespaceName',
		key: 'namespaceName',
		ellipsis: true,
		width: 130,
		sorter: false,
		align: 'left',
	},
	{
		title: <div>{t('col_successful').toString()}</div>,
		dataIndex: 'successful_pods',
		key: 'successful_pods',
		ellipsis: true,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_failed').toString()}</div>,
		dataIndex: 'failed_pods',
		key: 'failed_pods',
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_desired_successful').toString()}</div>,
		dataIndex: 'desired_successful_pods',
		key: 'desired_successful_pods',
		ellipsis: true,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_active').toString()}</div>,
		dataIndex: 'active_pods',
		key: 'active_pods',
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_cpu_req_usage_pct').toString()}</div>,
		dataIndex: 'cpu_request',
		key: 'cpu_request',
		width: 180,
		ellipsis: true,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_cpu_limit_usage_pct').toString()}</div>,
		dataIndex: 'cpu_limit',
		key: 'cpu_limit',
		width: 180,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_cpu_usage_cores').toString()}</div>,
		dataIndex: 'cpu',
		key: 'cpu',
		width: 170,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_mem_req_usage_pct').toString()}</div>,
		dataIndex: 'memory_request',
		key: 'memory_request',
		width: 165,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_mem_limit_usage_pct').toString()}</div>,
		dataIndex: 'memory_limit',
		key: 'memory_limit',
		width: 180,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_mem_usage_wss').toString()}</div>,
		dataIndex: 'memory',
		key: 'memory',
		width: 155,
		ellipsis: true,
		sorter: true,
		align: 'left',
	},
];

export const k8sJobsRenderRowData = (
	job: K8sJobsData,
	groupBy: BaseAutocompleteData[],
): K8sRenderedRowData => ({
	key: getRowKey(job, () => job.jobName || job.meta.k8s_job_name || '', groupBy),
	itemKey: job.meta.k8s_job_name,
	jobName: (
		<Tooltip title={job.meta.k8s_job_name}>{job.meta.k8s_job_name || ''}</Tooltip>
	),
	namespaceName: (
		<Tooltip title={job.meta.k8s_namespace_name}>
			{job.meta.k8s_namespace_name || ''}
		</Tooltip>
	),
	cpu_request: (
		<ValidateColumnValueWrapper
			value={job.cpuRequest}
			entity={InfraMonitoringEntity.JOBS}
			attribute="CPU Request"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={job.cpuRequest} type="request" />
			</div>
		</ValidateColumnValueWrapper>
	),
	cpu_limit: (
		<ValidateColumnValueWrapper
			value={job.cpuLimit}
			entity={InfraMonitoringEntity.JOBS}
			attribute="CPU Limit"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={job.cpuLimit} type="limit" />
			</div>
		</ValidateColumnValueWrapper>
	),
	cpu: (
		<ValidateColumnValueWrapper value={job.cpuUsage}>
			{job.cpuUsage}
		</ValidateColumnValueWrapper>
	),
	memory_request: (
		<ValidateColumnValueWrapper
			value={job.memoryRequest}
			entity={InfraMonitoringEntity.JOBS}
			attribute="Memory Request"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={job.memoryRequest} type="request" />
			</div>
		</ValidateColumnValueWrapper>
	),
	memory_limit: (
		<ValidateColumnValueWrapper
			value={job.memoryLimit}
			entity={InfraMonitoringEntity.JOBS}
			attribute="Memory Limit"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={job.memoryLimit} type="limit" />
			</div>
		</ValidateColumnValueWrapper>
	),
	memory: (
		<ValidateColumnValueWrapper value={job.memoryUsage}>
			{formatBytes(job.memoryUsage)}
		</ValidateColumnValueWrapper>
	),
	successful_pods: (
		<ValidateColumnValueWrapper value={job.successfulPods}>
			{job.successfulPods}
		</ValidateColumnValueWrapper>
	),
	desired_successful_pods: (
		<ValidateColumnValueWrapper value={job.desiredSuccessfulPods}>
			{job.desiredSuccessfulPods}
		</ValidateColumnValueWrapper>
	),
	failed_pods: (
		<ValidateColumnValueWrapper value={job.failedPods}>
			{job.failedPods}
		</ValidateColumnValueWrapper>
	),
	active_pods: (
		<ValidateColumnValueWrapper value={job.activePods}>
			{job.activePods}
		</ValidateColumnValueWrapper>
	),
	jobGroup: getGroupByEl(job, groupBy),
	...job.meta,
	groupedByMeta: getGroupedByMeta(job, groupBy),
});
