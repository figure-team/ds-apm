import type { TableColumnsType as ColumnsType } from 'antd';
import { TFunction } from 'i18next';
import { ServicesList } from 'types/api/metrics/getService';

import {
	ColumnKey,
	ColumnTitleKey,
	ColumnWidth,
	SORTING_ORDER,
} from './ColumnContants';

export const getBaseColumnOptions = (
	t: TFunction,
): ColumnsType<ServicesList> => [
	{
		title: t(ColumnTitleKey[ColumnKey.Application]).toString(),
		dataIndex: ColumnKey.Application,
		width: ColumnWidth.Application,
		key: ColumnKey.Application,
	},
	{
		dataIndex: ColumnKey.P99,
		key: ColumnKey.P99,
		width: ColumnWidth.P99,
		defaultSortOrder: SORTING_ORDER,
	},
	{
		title: t(ColumnTitleKey[ColumnKey.ErrorRate]).toString(),
		dataIndex: ColumnKey.ErrorRate,
		key: ColumnKey.ErrorRate,
		width: 150,
	},
	{
		title: t(ColumnTitleKey[ColumnKey.Operations]).toString(),
		dataIndex: ColumnKey.Operations,
		key: ColumnKey.Operations,
		width: ColumnWidth.Operations,
	},
];
