import { TableColumnType as ColumnType, Tooltip } from 'antd';
import { TFunction } from 'i18next';
import { Group } from 'lucide-react';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';

import { K8sRenderedRowData } from '../Base/types';
import { IEntityColumn } from '../Base/useInfraMonitoringTableColumnsStore';
import { getGroupByEl, getGroupedByMeta, getRowKey } from '../Base/utils';
import { formatBytes, ValidateColumnValueWrapper } from '../commonUtils';
import { K8sNodeData, K8sNodesListPayload } from './api';

import styles from './table.module.scss';

export interface K8sNodesRowData {
	key: string;
	itemKey: string;
	nodeUID: string;
	nodeName: React.ReactNode;
	clusterName: string;
	cpu: React.ReactNode;
	cpu_allocatable: React.ReactNode;
	memory: React.ReactNode;
	memory_allocatable: React.ReactNode;
	groupedByMeta?: any;
}

export const k8sNodesColumns: IEntityColumn[] = [
	{
		label: 'Node Group',
		value: 'nodeGroup',
		id: 'nodeGroup',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-collapse',
	},
	{
		label: 'Node Name',
		value: 'nodeName',
		id: 'nodeName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-expand',
	},
	{
		label: 'Cluster Name',
		value: 'clusterName',
		id: 'clusterName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
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

export const getK8sNodesListQuery = (): K8sNodesListPayload => ({
	filters: {
		items: [],
		op: 'and',
	},
	orderBy: { columnName: 'cpu', order: 'desc' },
});

export const k8sNodesColumnsConfig = (
	t: TFunction,
): ColumnType<K8sRenderedRowData>[] => [
	{
		title: (
			<div className={styles.entityGroupHeader}>
				<Group size={14} /> {t('col_node_group').toString()}
			</div>
		),
		dataIndex: 'nodeGroup',
		key: 'nodeGroup',
		ellipsis: true,
		width: 150,
		align: 'left',
		sorter: false,
	},
	{
		title: <div>{t('col_node_name').toString()}</div>,
		dataIndex: 'nodeName',
		key: 'nodeName',
		ellipsis: true,
		width: 80,
		sorter: false,
		align: 'left',
	},
	{
		title: <div>{t('col_cluster_name').toString()}</div>,
		dataIndex: 'clusterName',
		key: 'clusterName',
		ellipsis: true,
		width: 80,
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

export const k8sNodesRenderRowData = (
	node: K8sNodeData,
	groupBy: BaseAutocompleteData[],
): K8sRenderedRowData => ({
	key: getRowKey(
		node,
		() => node.nodeUID || node.meta.k8s_node_uid || node.meta.k8s_node_name,
		groupBy,
	),
	itemKey: node.meta.k8s_node_name,
	nodeUID: node.nodeUID || node.meta.k8s_node_uid,
	nodeName: (
		<Tooltip title={node.meta.k8s_node_name}>
			{node.meta.k8s_node_name || ''}
		</Tooltip>
	),
	clusterName: node.meta.k8s_cluster_name,
	cpu: (
		<ValidateColumnValueWrapper value={node.nodeCPUUsage}>
			{node.nodeCPUUsage}
		</ValidateColumnValueWrapper>
	),
	memory: (
		<ValidateColumnValueWrapper value={node.nodeMemoryUsage}>
			{formatBytes(node.nodeMemoryUsage)}
		</ValidateColumnValueWrapper>
	),
	cpu_allocatable: (
		<ValidateColumnValueWrapper value={node.nodeCPUAllocatable}>
			{node.nodeCPUAllocatable}
		</ValidateColumnValueWrapper>
	),
	memory_allocatable: (
		<ValidateColumnValueWrapper value={node.nodeMemoryAllocatable}>
			{formatBytes(node.nodeMemoryAllocatable)}
		</ValidateColumnValueWrapper>
	),
	nodeGroup: getGroupByEl(node, groupBy),
	...node.meta,
	groupedByMeta: getGroupedByMeta(node, groupBy),
});
