import { act, render, screen } from '@testing-library/react';
import { getToolTipValue } from 'components/Graph/yAxisConfig';
import TimezoneProvider from 'providers/Timezone';

import EChartsTimeSeries from '../components/EChartsTimeSeries';

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

import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import { LegendPosition } from 'lib/uPlotV2/components/types';

import echarts from '../echartsCore';

const apiResponse = {
	data: {
		result: [
			{ metric: {}, queryName: 'A', legend: '', values: [[1700000000, '1']] },
		],
		resultType: 'matrix',
	},
} as never;

function renderChart(): void {
	render(
		<TimezoneProvider>
			<EChartsTimeSeries
				widget={{ id: 'w1', thresholds: [], customLegendColors: {} } as never}
				chartData={[[1700000000], [1]] as never}
				configBuilder={new UPlotConfigBuilder({ id: 'w1' } as never)}
				apiResponse={apiResponse}
				isDarkMode
				timezone={{ name: 'Asia/Seoul', value: 'Asia/Seoul' } as never}
				legendPosition={LegendPosition.BOTTOM}
				onDragSelect={jest.fn()}
				onEngineError={jest.fn()}
				width={400}
				height={300}
			/>
		</TimezoneProvider>,
	);
}

describe('EChartsTimeSeries', () => {
	it('차트 컨테이너와 범례가 함께 렌더된다', () => {
		renderChart();
		expect(screen.getByTestId('echarts-cartesian')).toBeInTheDocument();
		expect(screen.getByTestId('legend')).toBeInTheDocument();
	});

	it('단일 시리즈 hover 시 TimeSeriesTooltip이 실제로 마운트되고 값이 표시된다 (리뷰 Critical #1 회귀 테스트)', async () => {
		renderChart();

		const initMock = echarts.init as jest.Mock;
		const instance = initMock.mock.results[initMock.mock.results.length - 1].value;

		// echartsCore.init()이 반환한 인스턴스의 on() 호출 중 'updateAxisPointer'
		// 리스너를 회수한다 — 실제 echarts 페이로드(axisTrigger.js)는 top-level
		// dataIndex 없이 axesInfo만 담으므로 그 형태로 직접 발화한다
		const onCalls = (instance.on as jest.Mock).mock.calls as Array<
			[string, (e: unknown) => void]
		>;
		const updateAxisPointerEntry = onCalls.find(
			([type]) => type === 'updateAxisPointer',
		);
		if (!updateAxisPointerEntry) {
			throw new Error('updateAxisPointer 리스너가 등록되지 않았다');
		}
		const [, updateAxisPointerHandler] = updateAxisPointerEntry;

		// getZr()는 mousemove/globalout 등록마다 각각 새 mock 객체를 반환하므로
		// (mock 팩토리가 매 호출 새 리터럴 생성) 모든 반환 객체를 순회해 mousemove
		// 핸들러를 찾는다. EChartsTooltipPositioner는 position이 null이면 포탈을
		// 렌더하지 않으므로(EChartsTooltipPositioner.tsx) hover.dataIndex뿐 아니라
		// mousePos도 채워야 TimeSeriesTooltip이 실제로 마운트된다.
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
		const [, mousemoveHandler] = mousemoveEntry;

		act(() => {
			updateAxisPointerHandler({
				axesInfo: [{ axisDim: 'x', value: 1700000000 * 1000 }],
			});
			mousemoveHandler({ event: { clientX: 120, clientY: 80 } });
		});

		// 헤더의 activeItem(단일 시리즈 값)이 실제로 렌더되어야 한다.
		// seriesIndex가 계속 null로 고정되면(리뷰 Critical #1) activeItem이 항상
		// null이라 이 testid 자체가 DOM에 존재하지 않는다.
		// mousePos는 rAF 스로틀로 다음 프레임에 반영되므로 findBy로 폴링한다.
		const expectedValueText = getToolTipValue(1, '', undefined);
		const content = await screen.findByTestId('uplot-tooltip-pinned-content');
		expect(content).toHaveTextContent(expectedValueText);
	});
});
