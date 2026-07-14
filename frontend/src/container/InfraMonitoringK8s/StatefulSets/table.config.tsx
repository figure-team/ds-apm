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
import { K8sStatefulSetsData } from './api';

import styles from './table.module.scss';

export const k8sStatefulSetsColumns: IEntityColumn[] = [
	{
		label: 'col_statefulset_group',
		value: 'statefulSetGroup',
		id: 'statefulSetGroup',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-collapse',
	},
	{
		label: 'col_statefulset_name',
		value: 'statefulsetName',
		id: 'statefulsetName',
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
		label: 'col_available',
		value: 'available_pods',
		id: 'available_pods',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_desired',
		value: 'desired_pods',
		id: 'desired_pods',
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

export const k8sStatefulSetsColumnsConfig = (
	t: TFunction,
): ColumnType<K8sRenderedRowData>[] => [
	{
		title: (
			<div className={styles.entityGroupHeader}>
				<Group size={14} /> {t('col_statefulset_group').toString()}
			</div>
		),
		dataIndex: 'statefulSetGroup',
		key: 'statefulSetGroup',
		ellipsis: true,
		width: 150,
		align: 'left',
		sorter: false,
	},
	{
		title: <div>{t('col_statefulset_name').toString()}</div>,
		dataIndex: 'statefulsetName',
		key: 'statefulsetName',
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
		title: <div>{t('col_available').toString()}</div>,
		dataIndex: 'available_pods',
		key: 'available_pods',
		width: 115,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_desired').toString()}</div>,
		dataIndex: 'desired_pods',
		key: 'desired_pods',
		width: 100,
		sorter: true,
		align: 'left',
	},
	{
		title: <div>{t('col_cpu_req_usage_pct').toString()}</div>,
		dataIndex: 'cpu_request',
		key: 'cpu_request',
		width: 165,
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
		sorter: true,
		align: 'left',
	},
];

export const k8sStatefulSetsRenderRowData = (
	statefulSet: K8sStatefulSetsData,
	groupBy: BaseAutocompleteData[],
): K8sRenderedRowData => ({
	key: getRowKey(
		statefulSet,
		() =>
			statefulSet.statefulSetName || statefulSet.meta.k8s_statefulset_name || '',
		groupBy,
	),
	itemKey: statefulSet.meta.k8s_statefulset_name,
	statefulsetName: (
		<Tooltip title={statefulSet.meta.k8s_statefulset_name}>
			{statefulSet.meta.k8s_statefulset_name || ''}
		</Tooltip>
	),
	namespaceName: (
		<Tooltip title={statefulSet.meta.k8s_namespace_name}>
			{statefulSet.meta.k8s_namespace_name || ''}
		</Tooltip>
	),
	cpu_request: (
		<ValidateColumnValueWrapper
			value={statefulSet.cpuRequest}
			entity={InfraMonitoringEntity.STATEFULSETS}
			attribute="CPU Request"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={statefulSet.cpuRequest} type="request" />
			</div>
		</ValidateColumnValueWrapper>
	),
	cpu_limit: (
		<ValidateColumnValueWrapper
			value={statefulSet.cpuLimit}
			entity={InfraMonitoringEntity.STATEFULSETS}
			attribute="CPU Limit"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={statefulSet.cpuLimit} type="limit" />
			</div>
		</ValidateColumnValueWrapper>
	),
	cpu: (
		<ValidateColumnValueWrapper value={statefulSet.cpuUsage}>
			{statefulSet.cpuUsage}
		</ValidateColumnValueWrapper>
	),
	memory_request: (
		<ValidateColumnValueWrapper
			value={statefulSet.memoryRequest}
			entity={InfraMonitoringEntity.STATEFULSETS}
			attribute="Memory Request"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={statefulSet.memoryRequest} type="request" />
			</div>
		</ValidateColumnValueWrapper>
	),
	memory_limit: (
		<ValidateColumnValueWrapper
			value={statefulSet.memoryLimit}
			entity={InfraMonitoringEntity.STATEFULSETS}
			attribute="Memory Limit"
		>
			<div className={styles.progressBar}>
				<EntityProgressBar value={statefulSet.memoryLimit} type="limit" />
			</div>
		</ValidateColumnValueWrapper>
	),
	memory: (
		<ValidateColumnValueWrapper value={statefulSet.memoryUsage}>
			{formatBytes(statefulSet.memoryUsage)}
		</ValidateColumnValueWrapper>
	),
	available_pods: (
		<ValidateColumnValueWrapper value={statefulSet.availablePods}>
			{statefulSet.availablePods}
		</ValidateColumnValueWrapper>
	),
	desired_pods: (
		<ValidateColumnValueWrapper value={statefulSet.desiredPods}>
			{statefulSet.desiredPods}
		</ValidateColumnValueWrapper>
	),
	statefulSetGroup: getGroupByEl(statefulSet, groupBy),
	...statefulSet.meta,
	groupedByMeta: getGroupedByMeta(statefulSet, groupBy),
});
