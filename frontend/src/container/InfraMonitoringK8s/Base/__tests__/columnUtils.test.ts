import { TableColumnType as ColumnType } from 'antd';

import {
	applyColumnDefaults,
	DEFAULT_COLUMN_WIDTH,
	DEFAULT_FIRST_COLUMN_WIDTH,
	EXPAND_ICON_COLUMN_WIDTH,
	getMinTableWidth,
} from '../columnUtils';
import { K8sRenderedRowData } from '../types';

type Col = ColumnType<K8sRenderedRowData>;

describe('applyColumnDefaults', () => {
	it('폭 미선언 첫 컬럼에 250px을 준다', () => {
		const columns: Col[] = [{ key: 'name' }, { key: 'cpu' }];

		const result = applyColumnDefaults(columns);

		expect(result[0].width).toBe(DEFAULT_FIRST_COLUMN_WIDTH);
	});

	it('폭 미선언 나머지 컬럼에 180px을 준다', () => {
		const columns: Col[] = [{ key: 'name' }, { key: 'cpu' }];

		const result = applyColumnDefaults(columns);

		expect(result[1].width).toBe(DEFAULT_COLUMN_WIDTH);
	});

	it('선언된 폭은 건드리지 않는다', () => {
		const columns: Col[] = [
			{ key: 'name', width: 300 },
			{ key: 'cpu', width: 80 },
		];

		const result = applyColumnDefaults(columns);

		expect(result[0].width).toBe(300);
		expect(result[1].width).toBe(80);
	});

	it('입력 배열과 원소를 변형하지 않는다', () => {
		const columns: Col[] = [{ key: 'name' }];

		const result = applyColumnDefaults(columns);

		expect(columns[0].width).toBeUndefined();
		expect(result).not.toBe(columns);
		expect(result[0]).not.toBe(columns[0]);
	});
});

describe('getMinTableWidth', () => {
	it('컬럼 폭을 합산한다', () => {
		const columns: Col[] = [
			{ key: 'name', width: 250 },
			{ key: 'cpu', width: 100 },
		];

		expect(getMinTableWidth(columns, false)).toBe(350);
	});

	it('확장 아이콘 컬럼이 있으면 40px을 더한다', () => {
		const columns: Col[] = [{ key: 'name', width: 250 }];

		expect(getMinTableWidth(columns, true)).toBe(250 + EXPAND_ICON_COLUMN_WIDTH);
	});

	it('숫자가 아닌 폭은 기본값으로 친다', () => {
		const columns: Col[] = [{ key: 'name', width: '50%' }];

		expect(getMinTableWidth(columns, false)).toBe(DEFAULT_COLUMN_WIDTH);
	});
});
