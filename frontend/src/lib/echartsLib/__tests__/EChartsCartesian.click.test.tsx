import { render } from '@testing-library/react';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';

import EChartsCartesian, {
	EChartsCartesianProps,
} from '../components/EChartsCartesian';

type ClickHandler = (params: unknown) => void;
type DownHandler = (e: { offsetX: number; offsetY: number }) => void;
let clickHandler: ClickHandler | null = null;
let downHandler: DownHandler | null = null;
const fakeZr = {
	on: (evt: string, fn: unknown): void => {
		if (evt === 'mousedown') downHandler = fn as DownHandler;
	},
	off: jest.fn(),
};
const fakeChart = {
	on: (evt: string, fn: ClickHandler): void => {
		if (evt === 'click') clickHandler = fn;
	},
	off: jest.fn(),
	getZr: () => fakeZr,
};
// EChartsView를 목으로 대체해 onInstanceReady에 가짜 인스턴스를 주입
jest.mock('../components/EChartsView', () => ({
	__esModule: true,
	default: (props: { onInstanceReady?: (c: unknown) => void }): JSX.Element => {
		props.onInstanceReady?.(fakeChart);
		return <div data-testid="view-mock" />;
	},
}));
// useEChartsEvents는 배선만 확인 — no-op 목(단, DRAG_CLICK_DIST_PX는 실제 값 유지)
jest.mock('../hooks/useEChartsEvents', () => ({
	__esModule: true,
	useEChartsEvents: jest.fn(),
	DRAG_CLICK_DIST_PX: 10,
}));

const apiResponse = {
	data: {
		result: [
			{ metric: { __name__: 'a' }, queryName: 'A', legend: '', values: [[1700000000, '10']] },
			{ metric: { __name__: 'b' }, queryName: 'B', legend: '', values: [[1700000000, '1']] },
		],
	},
} as never;
const chartData = [[1700000000], [10], [1]] as never;

// 스텁 buildOption — 이 테스트는 클릭 seam(seriesId→시리즈 특정)만 검증한다.
// 실제 막대 option은 barOption.test(Task 3)·EChartsBar.test(Task 4)가 담당.
const stubBuild = (): { option: never; seriesLabels: string[] } => ({
	option: { series: [] } as never,
	seriesLabels: ['a', 'b'],
});

const props: EChartsCartesianProps = {
	widget: { id: 'w1', customLegendColors: {} } as never,
	chartData,
	// ChartLayout이 실제로 config.getLegendItems()를 호출하므로(브리프의 `{} as
	// never` 스텁으로는 TypeError) 1단계 테스트(EChartsTimeSeries.test.tsx)와
	// 동일하게 실제 UPlotConfigBuilder 인스턴스를 사용한다.
	configBuilder: new UPlotConfigBuilder({ id: 'w1' } as never),
	apiResponse,
	currentQuery: undefined as never,
	isDarkMode: true,
	timezone: { value: 'UTC' } as never,
	legendPosition: 'bottom' as never,
	minTimeScale: 1700000000,
	maxTimeScale: 1700000000,
	onDragSelect: jest.fn(),
	onClick: jest.fn(),
	width: 400,
	height: 300,
	onEngineError: jest.fn(),
	buildOption: stubBuild,
	clickMode: 'bar',
};

describe('EChartsCartesian 막대 클릭 seam', () => {
	beforeEach(() => {
		clickHandler = null;
		downHandler = null;
		(props.onClick as jest.Mock).mockClear();
	});

	it('시리즈 click의 seriesId로 두 번째 시리즈를 특정한다', () => {
		render(<EChartsCartesian {...props} />);
		expect(clickHandler).not.toBeNull();
		clickHandler?.({
			seriesId: '1:b', // index 1 → seriesIndex 2
			dataIndex: 0,
			event: { offsetX: 100, offsetY: 50, event: { clientX: 100, clientY: 50 } },
		});
		const onClick = props.onClick as jest.Mock;
		expect(onClick).toHaveBeenCalledTimes(1);
		// focused(10번째 인자) seriesIndex=2, value=1
		expect(onClick.mock.calls[0][9]).toMatchObject({ seriesIndex: 2, value: 1 });
	});

	it('드래그 줌 제스처(10px↑ 이동)는 클릭을 억제한다', () => {
		render(<EChartsCartesian {...props} />);
		downHandler?.({ offsetX: 40, offsetY: 50 });
		clickHandler?.({
			seriesId: '1:b',
			dataIndex: 0,
			event: { offsetX: 100, offsetY: 50, event: { clientX: 100, clientY: 50 } },
		});
		expect(props.onClick as jest.Mock).not.toHaveBeenCalled();
	});
});
