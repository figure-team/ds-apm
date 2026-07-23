import { render } from '@testing-library/react';

import EChartsBar from '../components/EChartsBar';

let captured: Record<string, unknown> = {};
jest.mock('../components/EChartsCartesian', () => ({
	__esModule: true,
	default: (props: Record<string, unknown>): JSX.Element => {
		captured = props;
		return <div data-testid="cartesian-mock" />;
	},
}));

const baseProps = {
	widget: { id: 'w1', customLegendColors: {}, stackedBarChart: true } as never,
	chartData: [[1700000000], [10]] as never,
	configBuilder: {} as never,
	apiResponse: { data: { result: [{ metric: {}, queryName: 'A', legend: '', values: [[1700000000, '10']] }] } } as never,
	currentQuery: undefined as never,
	isDarkMode: true,
	timezone: { value: 'UTC' } as never,
	legendPosition: 'bottom' as never,
	onDragSelect: jest.fn(),
	width: 400,
	height: 300,
	onEngineError: jest.fn(),
};

describe('EChartsBar', () => {
	it('clickMode=bar로 EChartsCartesian을 렌더한다', () => {
		render(<EChartsBar {...baseProps} />);
		expect(captured.clickMode).toBe('bar');
	});

	it('buildOption이 막대 시리즈(type=bar)를 생성한다', () => {
		render(<EChartsBar {...baseProps} />);
		const build = captured.buildOption as (ctx: {
			visibilityMap: Record<number, boolean>;
			reducedMotion: boolean;
		}) => { option: { series: Array<{ type: string; stack?: string }> } };
		const { option } = build({ visibilityMap: {}, reducedMotion: false });
		expect(option.series[0].type).toBe('bar');
		expect(option.series[0].stack).toBe('total'); // stackedBarChart=true
	});
});
