import { TableColumnType as ColumnType } from 'antd';

import { K8sRenderedRowData } from './types';

export const DEFAULT_COLUMN_WIDTH = 180;
export const DEFAULT_FIRST_COLUMN_WIDTH = 250;
export const EXPAND_ICON_COLUMN_WIDTH = 40;

/**
 * 폭을 선언하지 않은 컬럼에만 기본값을 채운다.
 * table-layout:fixed 와 숫자 scroll.x 는 모든 표시 컬럼에 숫자 폭이 있어야 성립한다.
 * 부모 표와 확장 행 표는 서로 다른 <Table> 이므로 양쪽 모두 이 함수를 통과시킨다.
 * (두 표의 컬럼 집합이 다르므로 이 함수가 둘의 폭을 정렬시켜 주지는 않는다.)
 * table.config 의 컬럼 배열은 모듈 레벨 상수이므로 입력을 변형하지 않는다.
 */
export function applyColumnDefaults(
	columns: ColumnType<K8sRenderedRowData>[],
): ColumnType<K8sRenderedRowData>[] {
	return columns.map((column, index) => {
		if (column.width !== undefined) {
			return column;
		}

		return {
			...column,
			width: index === 0 ? DEFAULT_FIRST_COLUMN_WIDTH : DEFAULT_COLUMN_WIDTH,
		};
	});
}

/**
 * scroll.x 에 넘길 표 최소 폭. 숫자를 주면 rc-table 이 표를
 * `width: <x>px; min-width: 100%` 로 잡아, 공간이 남으면 채우고 모자랄 때만 스크롤한다.
 */
export function getMinTableWidth(
	columns: ColumnType<K8sRenderedRowData>[],
	hasExpandColumn: boolean,
): number {
	const columnsWidth = columns.reduce(
		(sum, column) =>
			sum +
			(typeof column.width === 'number' ? column.width : DEFAULT_COLUMN_WIDTH),
		0,
	);

	return columnsWidth + (hasExpandColumn ? EXPAND_ICON_COLUMN_WIDTH : 0);
}
