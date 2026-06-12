import { TableColumnType as ColumnType, Tooltip } from 'antd';
import { TFunction } from 'i18next';
import { Group } from 'lucide-react';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';

import { K8sRenderedRowData } from '../Base/types';
import { IEntityColumn } from '../Base/useInfraMonitoringTableColumnsStore';
import { getGroupByEl, getGroupedByMeta, getRowKey } from '../Base/utils';
import { formatBytes, ValidateColumnValueWrapper } from '../commonUtils';
import { K8sClusterData, K8sClustersListPayload } from './api';

import styles from './table.module.scss';

export interface K8sClustersRowData {
	key: string;
	itemKey: string;
	clusterUID: string;
	clusterName: React.ReactNode;
	cpu: React.ReactNode;
	cpu_allocatable: React.ReactNode;
	memory: React.ReactNode;
	memory_allocatable: React.ReactNode;
	groupedByMeta?: Record<string, string>;
}

export const k8sClustersColumns: IEntityColumn[] = [
	{
		label: 'Cluster Group',
		value: 'clusterGroup',
		id: 'clusterGroup',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-collapse',
	},
	{
		label: 'Cluster Name',
		value: 'clusterName',
		id: 'clusterName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-expand',
	},
	{
		label: 'CPU Usage (cores)',
		value: 'cpu',
		id: 'cpu',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'CPU Alloc (cores)',
		value: 'cpu_allocatable',
		id: 'cpu_allocatable',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'Memory Usage (WSS)',
		value: 'memory',
		id: 'memory',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'Memory Alloc (bytes)',
		value: 'memory_allocatable',
		id: 'memory_allocatable',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
];

export const getK8sClustersListQuery = (): K8sClustersListPayload => ({
	filters: {
		items: [],
		op: 'and',
	},
	orderBy: { columnName: 'cpu', order: 'desc' },
});

export const k8sClustersColumnsConfig = (
	t: TFunction,
): ColumnType<K8sRenderedRowData>[] => [
	{
		title: (
			<div className={styles.entityGroupHeader}>
				<Group size={14} /> {t('col_cluster_group').toString()}
			</div>
		),
		dataIndex: 'clusterGroup',
		key: 'clusterGroup',
		ellipsis: true,
		width: 150,
		align: 'left',
		sorter: false,
	},
	{
		title: <div>{t('col_cluster_name').toString()}</div>,
		dataIndex: 'clusterName',
		key: 'clusterName',
		ellipsis: true,
		width: 150,
		sorter: false,
		align: 'left',
	},
	{
		title: <div>{t('col_cpu_usage_cores').toString()}</div>,
		dataIndex: 'cpu',
		key: 'cpu',
		width: 80,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_cpu_alloc_cores').toString()}</div>,
		dataIndex: 'cpu_allocatable',
		key: 'cpu_allocatable',
		width: 80,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_memory_usage_wss').toString()}</div>,
		dataIndex: 'memory',
		key: 'memory',
		width: 80,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_memory_allocatable').toString()}</div>,
		dataIndex: 'memory_allocatable',
		key: 'memory_allocatable',
		width: 80,
		sorter: true,
		align: 'left',
	},
];

export const k8sClustersRenderRowData = (
	cluster: K8sClusterData,
	groupBy: BaseAutocompleteData[],
): K8sRenderedRowData => ({
	key: getRowKey(
		cluster,
		() =>
			cluster.clusterUID ||
			cluster.meta.k8s_cluster_uid ||
			cluster.meta.k8s_cluster_name,
		groupBy,
	),
	itemKey: cluster.meta.k8s_cluster_name,
	clusterUID: cluster.clusterUID || cluster.meta.k8s_cluster_uid,
	clusterName: (
		<Tooltip title={cluster.meta.k8s_cluster_name}>
			{cluster.meta.k8s_cluster_name || ''}
		</Tooltip>
	),
	cpu: (
		<ValidateColumnValueWrapper value={cluster.cpuUsage}>
			{cluster.cpuUsage}
		</ValidateColumnValueWrapper>
	),
	memory: (
		<ValidateColumnValueWrapper value={cluster.memoryUsage}>
			{formatBytes(cluster.memoryUsage)}
		</ValidateColumnValueWrapper>
	),
	cpu_allocatable: (
		<ValidateColumnValueWrapper value={cluster.cpuAllocatable}>
			{cluster.cpuAllocatable}
		</ValidateColumnValueWrapper>
	),
	memory_allocatable: (
		<ValidateColumnValueWrapper value={cluster.memoryAllocatable}>
			{formatBytes(cluster.memoryAllocatable)}
		</ValidateColumnValueWrapper>
	),
	clusterGroup: getGroupByEl(cluster, groupBy),
	...cluster.meta,
	groupedByMeta: getGroupedByMeta(cluster, groupBy),
});
