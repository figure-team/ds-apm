import { act, render, screen } from '@testing-library/react';
import { getToolTipValue } from 'components/Graph/yAxisConfig';
import TimezoneProvider from 'providers/Timezone';

import EChartsHistogram from '../components/EChartsHistogram';

jest.mock('../echartsCore', () => ({
	__esModule: true,
	default: {
		init: jest.fn(() => ({
			setOption: jest.fn(),
			resize: jest.fn(),
			dispose: jest.fn(),
			on: jest.fn(),
			off: jest.fn(),
			getZr: jest.fn(() => ({ on: jest.fn(), off: jest.fn() })),
			dispatchAction: jest.fn(),
			getOption: jest.fn(() => ({ legend: [{ selected: {} }] })),
		})),
		registerTheme: jest.fn(),
	},
}));

// Legend는 UPlotConfigBuilder 계약 검증이 목적이 아니므로 표시 여부만 확인
jest.mock('lib/uPlotV2/components/Legend/Legend', () => ({
	__esModule: true,
	default: (): JSX.Element => <div data-testid="legend" />,
}));

import { LegendPosition } from 'lib/uPlotV2/components/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';

import echarts from '../echartsCore';

// 유령 널빈(-10) + 버킷 0,10,20. bucketSize=10 → 중심은 -5, 5, 15, 25
const chartData = [
	[-10, 0, 10, 20],
	[null, 5, 12, 4],
] as never;

const apiResponse = {
	data: {
		result: [{ metric: { __name__: 'a' }, queryName: 'A', legend: '', values: [] }],
		resultType: 'matrix',
	},
} as never;

// I-1 회귀 데이터: 표시 시리즈 2개('a','b') · 버킷 index2(x=15)에서
// 'a'=12(유한)·'b'=null(비기여) → "기여 시리즈 1개" 케이스를 재현한다
const twoSeriesChartData = [
	[-10, 0, 10, 20],
	[null, 5, 12, 4],
	[null, 3, null, 6],
] as never;

const twoSeriesApiResponse = {
	data: {
		result: [
			{ metric: { __name__: 'a' }, queryName: 'A', legend: '', values: [] },
			{ metric: { __name__: 'b' }, queryName: 'B', legend: '', values: [] },
		],
		resultType: 'matrix',
	},
} as never;

function renderChart(
	overrides: { chartData?: unknown; apiResponse?: unknown } = {},
): void {
	render(
		<TimezoneProvider>
			<EChartsHistogram
				widget={{ id: 'w1', customLegendColors: {}, yAxisUnit: 'none' } as never}
				chartData={(overrides.chartData ?? chartData) as never}
				configBuilder={new UPlotConfigBuilder({ id: 'w1' } as never)}
				apiResponse={(overrides.apiResponse ?? apiResponse) as never}
				isDarkMode
				legendPosition={LegendPosition.BOTTOM}
				isQueriesMerged={false}
				canPinTooltip
				width={400}
				height={300}
				onEngineError={jest.fn()}
			/>
		</TimezoneProvider>,
	);
}

/** 최근 init() 인스턴스에서 chart/zr 리스너를 회수한다 (2a 패턴) */
function grabListeners(): {
	fireAxisPointer: (value: number) => void;
	fireMouseMove: () => void;
} {
	const initMock = echarts.init as jest.Mock;
	const instance = initMock.mock.results[initMock.mock.results.length - 1].value;

	const onCalls = (instance.on as jest.Mock).mock.calls as Array<
		[string, (e: unknown) => void]
	>;
	const axisPointerEntry = onCalls.find(([type]) => type === 'updateAxisPointer');
	if (!axisPointerEntry) {
		throw new Error('updateAxisPointer 리스너가 등록되지 않았다');
	}

	// getZr()는 호출마다 새 mock 객체를 반환하므로 전 반환값을 순회한다
	const zrInstances = (instance.getZr as jest.Mock).mock.results.map(
		(r) => r.value,
	);
	const mousemoveEntry = zrInstances
		.flatMap(
			(zr) => (zr.on as jest.Mock).mock.calls as Array<[string, (e: unknown) => void]>,
		)
		.find(([type]) => type === 'mousemove');
	if (!mousemoveEntry) {
		throw new Error('zr mousemove 리스너가 등록되지 않았다');
	}

	return {
		fireAxisPointer: (value): void =>
			axisPointerEntry[1]({ axesInfo: [{ axisDim: 'x', value }] }),
		// Positioner는 position이 null이면 포탈을 렌더하지 않는다
		fireMouseMove: (): void =>
			mousemoveEntry[1]({ event: { clientX: 120, clientY: 80 } }),
	};
}

describe('EChartsHistogram', () => {
	it('차트 컨테이너와 범례가 함께 렌더된다', () => {
		renderChart();
		expect(screen.getByTestId('echarts-histogram-view')).toBeInTheDocument();
		expect(screen.getByTestId('legend')).toBeInTheDocument();
	});

	it('스냅된 버킷 중심 hover가 그 버킷의 카운트를 표시한다 (R2)', async () => {
		renderChart();
		const { fireAxisPointer, fireMouseMove } = grabListeners();
		// x=15는 [10,20) 중심 → index 2 → count 12
		// (bracket이 아니라 최근접 "엣지"였다면 index 3 = 4가 나왔을 것)
		act(() => {
			fireAxisPointer(15);
			fireMouseMove();
		});
		// 단일 시리즈라 showList=false → activeItem(헤더)로 값이 표시된다
		const content = await screen.findByTestId('uplot-tooltip-pinned-content');
		expect(content).toHaveTextContent(getToolTipValue(12, '', undefined));
	});

	it('선행 유령 널빈 영역 hover는 툴팁을 띄우지 않는다 (R3)', () => {
		renderChart();
		const { fireAxisPointer, fireMouseMove } = grabListeners();
		// x=-5는 유령 널빈([-10,0)) 중심 → skipLeadingNullBin으로 null
		act(() => {
			fireAxisPointer(-5);
			fireMouseMove();
		});
		expect(screen.queryByTestId('uplot-tooltip-container')).not.toBeInTheDocument();
	});

	it('표시 시리즈 2개 중 해당 버킷에서 유한값을 가진 시리즈가 1개뿐이면 그 라벨·카운트가 표시된다 (I-1)', async () => {
		renderChart({ chartData: twoSeriesChartData, apiResponse: twoSeriesApiResponse });
		const { fireAxisPointer, fireMouseMove } = grabListeners();
		// x=15 → 버킷 index2: 'a'=12(유한) · 'b'=null(비기여) → 기여 시리즈는 'a' 하나뿐
		act(() => {
			fireAxisPointer(15);
			fireMouseMove();
		});
		const content = await screen.findByTestId('uplot-tooltip-pinned-content');
		expect(content).toHaveTextContent('a');
		expect(content).toHaveTextContent(getToolTipValue(12, '', undefined));
	});
});
