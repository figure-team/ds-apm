import { render } from '@testing-library/react';
import { usePlotContext } from 'lib/uPlotV2/context/PlotContext';

import EChartsPlotContextProvider from '../context/EChartsPlotContextProvider';

// legendVisibilityUtils는 localStorage I/O를 수반하므로 모킹해 호출 계약만 검증한다
jest.mock(
	'container/DashboardContainer/visualization/panels/utils/legendVisibilityUtils',
	() => ({
		updateSeriesVisibilityToLocalStorage: jest.fn(),
	}),
);

// eslint-disable-next-line @typescript-eslint/no-var-requires
const { updateSeriesVisibilityToLocalStorage } = jest.requireMock(
	'container/DashboardContainer/visualization/panels/utils/legendVisibilityUtils',
);

const dispatchAction = jest.fn();
const fakeChart = { dispatchAction } as never;
// 리뷰 반영: 훅 수동 발화 검증용 — getConfig().hooks.setSeries 참조 계약
const setSeriesHook = jest.fn();
const fakeConfig = {
	getConfig: () => ({ hooks: { setSeries: [setSeriesHook] } }),
} as never;
const onVisibilityChange = jest.fn();

function Probe({ onReady }: { onReady: (ctx: ReturnType<typeof usePlotContext>) => void }): null {
	onReady(usePlotContext());
	return null;
}

describe('EChartsPlotContextProvider', () => {
	beforeEach(() => jest.clearAllMocks());

	function renderWithCtx(
		overrides: Partial<{ shouldSaveSelectionPreference: boolean; widgetId: string }> = {},
	): ReturnType<typeof usePlotContext> {
		let ctx: ReturnType<typeof usePlotContext> | undefined;
		render(
			<EChartsPlotContextProvider
				chart={fakeChart}
				widgetId={overrides.widgetId ?? 'w1'}
				seriesLabels={['series-A', 'series-B']}
				config={fakeConfig}
				onVisibilityChange={onVisibilityChange}
				shouldSaveSelectionPreference={overrides.shouldSaveSelectionPreference ?? false}
			>
				<Probe onReady={(c): void => { ctx = c; }} />
			</EChartsPlotContextProvider>,
		);
		if (!ctx) throw new Error('context not provided');
		return ctx;
	}

	it('onToggleSeriesOnOff는 표시 상태를 뒤집고 setSeries 훅·onVisibilityChange로 전파한다', () => {
		renderWithCtx().onToggleSeriesOnOff(1);
		expect(setSeriesHook).toHaveBeenCalledWith(null, 1, { show: false });
		expect(onVisibilityChange).toHaveBeenCalledWith(
			expect.objectContaining({ 1: false }),
		);
	});

	it('onToggleSeriesVisibility(솔로)는 대상만 남기고, 재클릭 시 전체 복원', () => {
		const ctx = renderWithCtx();
		ctx.onToggleSeriesVisibility(2);
		expect(onVisibilityChange).toHaveBeenLastCalledWith({ 1: false, 2: true });
		ctx.onToggleSeriesVisibility(2);
		expect(onVisibilityChange).toHaveBeenLastCalledWith({ 1: true, 2: true });
	});

	it('onFocusSeries는 highlight/downplay를 디스패치한다', () => {
		const ctx = renderWithCtx();
		ctx.onFocusSeries(1);
		expect(dispatchAction).toHaveBeenCalledWith({
			type: 'highlight',
			seriesName: 'series-A',
		});
		ctx.onFocusSeries(null);
		expect(dispatchAction).toHaveBeenCalledWith({ type: 'downplay' });
	});

	it('syncSeriesVisibilityToLocalStorage는 [0]=Timestamp 규약의 items로 저장 유틸을 호출한다', () => {
		const ctx = renderWithCtx({ widgetId: 'w-sync' });
		ctx.onToggleSeriesOnOff(2); // series-B 숨김 상태로 만든 뒤 수동 동기화 호출
		ctx.syncSeriesVisibilityToLocalStorage();
		expect(updateSeriesVisibilityToLocalStorage).toHaveBeenCalledWith('w-sync', [
			{ label: 'Timestamp', show: true },
			{ label: 'series-A', show: true },
			{ label: 'series-B', show: false },
		]);
	});

	it('shouldSaveSelectionPreference=true면 토글 시 저장을 트리거하고 false면 저장하지 않는다', () => {
		const ctxOn = renderWithCtx({
			shouldSaveSelectionPreference: true,
			widgetId: 'w-save-on',
		});
		ctxOn.onToggleSeriesOnOff(1);
		expect(updateSeriesVisibilityToLocalStorage).toHaveBeenCalledWith(
			'w-save-on',
			expect.any(Array),
		);

		(updateSeriesVisibilityToLocalStorage as jest.Mock).mockClear();

		const ctxOff = renderWithCtx({
			shouldSaveSelectionPreference: false,
			widgetId: 'w-save-off',
		});
		ctxOff.onToggleSeriesOnOff(1);
		expect(updateSeriesVisibilityToLocalStorage).not.toHaveBeenCalled();
	});
});
