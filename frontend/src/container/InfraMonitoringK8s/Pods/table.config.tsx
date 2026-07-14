import React from 'react';
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
import { K8sPodsData } from './api';

import styles from './table.module.scss';

export interface K8sPodsRowData {
	key: string;
	podName: React.ReactNode;
	podUID: string;
	cpu_request: React.ReactNode;
	cpu_limit: React.ReactNode;
	cpu: React.ReactNode;
	memory_request: React.ReactNode;
	memory_limit: React.ReactNode;
	memory: React.ReactNode;
	restarts: React.ReactNode;
	groupedByMeta?: any;
}

export const k8sPodColumns: IEntityColumn[] = [
	{
		label: 'col_pod_group',
		value: 'podGroup',
		id: 'podGroup',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-collapse',
	},
	{
		label: 'col_pod_name',
		value: 'podName',
		id: 'podName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-expand',
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
	{
		label: 'col_namespace_name',
		value: 'namespace',
		id: 'namespace',
		canBeHidden: true,
		defaultVisibility: false,
		behavior: 'always-visible',
	},
	{
		label: 'col_node_name',
		value: 'node',
		id: 'node',
		canBeHidden: true,
		defaultVisibility: false,
		behavior: 'always-visible',
	},
	{
		label: 'col_cluster_name',
		value: 'cluster',
		id: 'cluster',
		canBeHidden: true,
		defaultVisibility: false,
		behavior: 'always-visible',
	},
	// TODO - Re-enable the column once backend issue is fixed
	// {
	// 	label: 'Restarts',
	// 	value: 'restarts',
	// 	id: 'restarts',
	// 	canRemove: false,
	// },
];

export const k8sPodColumnsConfig = (
	t: TFunction,
): ColumnType<K8sRenderedRowData>[] => [
	{
		title: (
			<div className={styles.entityGroupHeader}>
				<Group size={14} /> {t('col_pod_group').toString()}
			</div>
		),
		dataIndex: 'podGroup',
		key: 'podGroup',
		ellipsis: true,
		width: 180,
		sorter: false,
	},
	{
		title: <div>{t('col_pod_name').toString()}</div>,
		dataIndex: 'podName',
		key: 'podName',
		width: 250,
		ellipsis: true,
		sorter: false,
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
	{
		title: <div>{t('col_namespace').toString()}</div>,
		dataIndex: 'namespace',
		key: 'namespace',
		width: 100,
		sorter: false,
		ellipsis: true,
		align: 'left',
	},
	{
		title: <div>{t('col_node').toString()}</div>,
		dataIndex: 'node',
		key: 'node',
		width: 100,
		sorter: false,
		ellipsis: true,
		align: 'left',
	},
	{
		title: <div>{t('col_cluster').toString()}</div>,
		dataIndex: 'cluster',
		key: 'cluster',
		width: 100,
		sorter: false,
		ellipsis: true,
		align: 'left',
	},
	// TODO - Re-enable the column once backend issue is fixed
	// {
	// 	title: (
	// 		<div className="column-header">
	// 			<Tooltip title="Container Restarts">Restarts</Tooltip>
	// 		</div>
	// 	),
	// 	dataIndex: 'restarts',
	// 	key: 'restarts',
	// 	width: 40,
	// 	ellipsis: true,
	// 	sorter: true,
	// 	align: 'left',
	// 	className: `column ${columnProgressBarClassName}`,
	// },
];

export const k8sPodRenderRowData = (
	pod: K8sPodsData,
	groupBy: BaseAutocompleteData[],
): K8sRenderedRowData => ({
	key: getRowKey(
		pod,
		() => pod.podUID || pod.meta.k8s_pod_uid || pod.meta.k8s_pod_name,
		groupBy,
	),
	itemKey: pod.podUID,
	podName: (
		<Tooltip title={pod.meta.k8s_pod_name || ''}>
			{pod.meta.k8s_pod_name || ''}
		</Tooltip>
	),
	podUID: pod.podUID || '',
	cpu_request: (
		<ValidateColumnValueWrapper
			value={pod.podCPURequest}
			entity={InfraMonitoringEntity.PODS}
			attribute="CPU Request"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={pod.podCPURequest} type="request" />
			</div>
		</ValidateColumnValueWrapper>
	),
	cpu_limit: (
		<ValidateColumnValueWrapper
			value={pod.podCPULimit}
			entity={InfraMonitoringEntity.PODS}
			attribute="CPU Limit"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={pod.podCPULimit} type="limit" />
			</div>
		</ValidateColumnValueWrapper>
	),
	cpu: (
		<ValidateColumnValueWrapper value={pod.podCPU}>
			{pod.podCPU}
		</ValidateColumnValueWrapper>
	),
	memory_request: (
		<ValidateColumnValueWrapper
			value={pod.podMemoryRequest}
			entity={InfraMonitoringEntity.PODS}
			attribute="Memory Request"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={pod.podMemoryRequest} type="request" />
			</div>
		</ValidateColumnValueWrapper>
	),
	memory_limit: (
		<ValidateColumnValueWrapper
			value={pod.podMemoryLimit}
			entity={InfraMonitoringEntity.PODS}
			attribute="Memory Limit"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={pod.podMemoryLimit} type="limit" />
			</div>
		</ValidateColumnValueWrapper>
	),
	memory: (
		<ValidateColumnValueWrapper value={pod.podMemory}>
			{formatBytes(pod.podMemory)}
		</ValidateColumnValueWrapper>
	),
	restarts: (
		<ValidateColumnValueWrapper value={pod.restartCount}>
			{pod.restartCount}
		</ValidateColumnValueWrapper>
	),
	namespace: pod.meta.k8s_namespace_name,
	node: pod.meta.k8s_node_name,
	cluster: pod.meta.k8s_cluster_name,
	meta: pod.meta,
	podGroup: getGroupByEl(pod, groupBy),
	...pod.meta,
	groupedByMeta: getGroupedByMeta(pod, groupBy),
});
