import React from 'react';
import { InfoCircleOutlined } from '@ant-design/icons';
import { Progress, TableColumnType as ColumnType, Tag, Tooltip } from 'antd';
import { HostData } from 'api/infraMonitoring/getHostLists';
import { K8sRenderedRowData } from 'container/InfraMonitoringK8s/Base/types';
import { IEntityColumn } from 'container/InfraMonitoringK8s/Base/useInfraMonitoringTableColumnsStore';
import {
	getGroupByEl,
	getGroupedByMeta,
	getRowKey,
} from 'container/InfraMonitoringK8s/Base/utils';
import { ValidateColumnValueWrapper } from 'container/InfraMonitoringK8s/commonUtils';
import { InfraMonitoringEntity } from 'container/InfraMonitoringK8s/constants';
import { TFunction } from 'i18next';
import { Group } from 'lucide-react';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';

import { getMemoryProgressColor, getProgressColor } from './constants';
import { HostnameCell } from './utils';

import styles from './table.module.scss';

export const hostColumns: IEntityColumn[] = [
	{
		label: 'col_host_group',
		value: 'hostGroup',
		id: 'hostGroup',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-collapse',
	},
	{
		label: 'col_hostname',
		value: 'hostName',
		id: 'hostName',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'hidden-on-expand',
	},
	{
		label: 'col_status',
		value: 'active',
		id: 'active',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_cpu_usage',
		value: 'cpu',
		id: 'cpu',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_memory_usage',
		value: 'memory',
		id: 'memory',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_iowait',
		value: 'wait',
		id: 'wait',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
	{
		label: 'col_load_avg',
		value: 'load15',
		id: 'load15',
		canBeHidden: false,
		defaultVisibility: true,
		behavior: 'always-visible',
	},
];

export const hostColumnsConfig = (
	t: TFunction,
): ColumnType<K8sRenderedRowData>[] => [
	{
		title: (
			<div className={styles.entityGroupHeader}>
				<Group size={14} /> {t('col_host_group').toString()}
			</div>
		),
		dataIndex: 'hostGroup',
		key: 'hostGroup',
		ellipsis: true,
		width: 180,
		sorter: false,
	},
	{
		title: (
			<div className={styles.hostnameColumnHeader}>
				{t('col_hostname').toString()}
			</div>
		),
		dataIndex: 'hostName',
		key: 'hostName',
		width: 250,
		render: (_value, record): React.ReactNode => (
			<HostnameCell
				hostName={typeof record.hostName === 'string' ? record.hostName : ''}
			/>
		),
	},
	{
		title: (
			<div className={styles.statusHeader}>
				{t('col_status').toString()}
				<Tooltip title={t('sent_system_metrics_last_10m').toString()}>
					<InfoCircleOutlined />
				</Tooltip>
			</div>
		),
		dataIndex: 'active',
		key: 'active',
		width: 100,
	},
	{
		title: (
			<div className={styles.columnHeaderRight}>
				{t('col_cpu_usage').toString()}
			</div>
		),
		dataIndex: 'cpu',
		key: 'cpu',
		width: 100,
		sorter: true,
		align: 'right',
	},
	{
		title: (
			<div className={`${styles.columnHeaderRight} ${styles.memoryUsageHeader}`}>
				{t('col_memory_usage').toString()}
				<Tooltip title={t('excluding_cache_memory').toString()}>
					<InfoCircleOutlined />
				</Tooltip>
			</div>
		),
		dataIndex: 'memory',
		key: 'memory',
		width: 100,
		sorter: true,
		align: 'right',
	},
	{
		title: (
			<div className={styles.columnHeaderRight}>{t('col_iowait').toString()}</div>
		),
		dataIndex: 'wait',
		key: 'wait',
		width: 100,
		sorter: true,
		align: 'right',
	},
	{
		title: (
			<div className={styles.columnHeaderRight}>
				{t('col_load_avg').toString()}
			</div>
		),
		dataIndex: 'load15',
		key: 'load15',
		width: 100,
		sorter: true,
		align: 'right',
	},
];

function hostRowSource(host: HostData): { meta: Record<string, string> } {
	return {
		meta: {
			...(host.meta ?? {}),
			host_name: host.hostName ?? '',
			'host.name': host.hostName ?? '',
			os_type: host.os ?? '',
			'os.type': host.os ?? '',
		},
	};
}

export const hostRenderRowData = (
	host: HostData,
	groupBy: BaseAutocompleteData[],
): K8sRenderedRowData => {
	const synthetic = hostRowSource(host);
	const rowKey = getRowKey(synthetic, () => host.hostName || 'unknown', groupBy);
	const groupedByMeta = getGroupedByMeta(synthetic, groupBy);
	const cpuPercent = Number((host.cpu * 100).toFixed(1));
	const memoryPercent = Number((host.memory * 100).toFixed(1));

	return {
		key: rowKey,
		itemKey: host.hostName ?? '',
		groupedByMeta,
		meta: synthetic.meta,
		hostGroup: getGroupByEl(synthetic, groupBy),
		...synthetic.meta,
		hostName: host.hostName ?? '',
		active: (
			<Tag
				bordered
				className={`${styles.statusTag} ${
					host.active ? styles.statusTagActive : styles.statusTagInactive
				}`}
			>
				{host.active ? 'ACTIVE' : 'INACTIVE'}
			</Tag>
		),
		cpu: (
			<div className={styles.progressContainer}>
				<ValidateColumnValueWrapper
					value={host.cpu}
					entity={InfraMonitoringEntity.HOSTS}
				>
					<Progress
						percent={cpuPercent}
						strokeLinecap="butt"
						size="small"
						strokeColor={getProgressColor(cpuPercent)}
						className={styles.progressBar}
					/>
				</ValidateColumnValueWrapper>
			</div>
		),
		memory: (
			<div className={styles.progressContainer}>
				<ValidateColumnValueWrapper
					value={host.memory}
					entity={InfraMonitoringEntity.HOSTS}
				>
					<Progress
						percent={memoryPercent}
						strokeLinecap="butt"
						size="small"
						strokeColor={getMemoryProgressColor(memoryPercent)}
						className={styles.progressBar}
					/>
				</ValidateColumnValueWrapper>
			</div>
		),
		wait: `${Number((host.wait * 100).toFixed(1))}%`,
		load15: host.load15,
	};
};
