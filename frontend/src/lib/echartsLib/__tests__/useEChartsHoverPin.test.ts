import { act, renderHook } from '@testing-library/react';

import { useEChartsHoverPin } from '../hooks/useEChartsHoverPin';

type Listener = (payload: unknown) => void;

function makeFakeChart(): {
	instance: never;
	fire: (evt: string, payload?: unknown) => void;
	zrFire: (evt: string, payload?: unknown) => void;
} {
	const chartListeners: Record<string, Listener> = {};
	const zrListeners: Record<string, Listener> = {};
	const instance = {
		on: (evt: string, fn: Listener): void => {
			chartListeners[evt] = fn;
		},
		getZr: () => ({
			on: (evt: string, fn: Listener): void => {
				zrListeners[evt] = fn;
			},
		}),
	} as never;
	return {
		instance,
		fire: (evt, payload): void => chartListeners[evt]?.(payload),
		zrFire: (evt, payload): void => zrListeners[evt]?.(payload),
	};
}

describe('useEChartsHoverPin', () => {
	beforeEach(() => {
		jest
			.spyOn(window, 'requestAnimationFrame')
			.mockImplementation((cb: FrameRequestCallback): number => {
				cb(0);
				return 1;
			});
	});
	afterEach(() => jest.restoreAllMocks());

	it('updateAxisPointer에서 주입된 resolveIndex로 dataIndex를 정한다', () => {
		const fake = makeFakeChart();
		const resolveIndex = jest.fn().mockReturnValue(7);
		const { result } = renderHook(() =>
			useEChartsHoverPin({ canPinTooltip: true, resolveIndex }),
		);
		act(() => result.current.handleInstanceReady(fake.instance));
		act(() => fake.fire('updateAxisPointer', { axesInfo: [] }));
		expect(resolveIndex).toHaveBeenCalled();
		expect(result.current.hover.dataIndex).toBe(7);
	});

	it("'p' 키로 핀 토글, Esc로 해제", () => {
		const fake = makeFakeChart();
		const { result } = renderHook(() =>
			useEChartsHoverPin({ canPinTooltip: true, resolveIndex: () => 3 }),
		);
		act(() => result.current.handleInstanceReady(fake.instance));
		act(() => fake.fire('updateAxisPointer', { axesInfo: [] }));
		act(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'p' }));
		});
		expect(result.current.hover.pinned).toBe(true);
		act(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
		});
		expect(result.current.hover.pinned).toBe(false);
	});

	it('핀 상태에선 dataIndex 갱신이 멈춘다', () => {
		const fake = makeFakeChart();
		let idx = 1;
		const { result } = renderHook(() =>
			useEChartsHoverPin({ canPinTooltip: true, resolveIndex: () => idx }),
		);
		act(() => result.current.handleInstanceReady(fake.instance));
		act(() => fake.fire('updateAxisPointer', {}));
		act(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'p' }));
		});
		idx = 99;
		act(() => fake.fire('updateAxisPointer', {}));
		expect(result.current.hover.dataIndex).toBe(1);
	});

	it('mousemove는 rAF로 좌표를 반영하고 핀 상태에선 건너뛴다', () => {
		const fake = makeFakeChart();
		const { result } = renderHook(() =>
			useEChartsHoverPin({ canPinTooltip: true, resolveIndex: () => 0 }),
		);
		act(() => result.current.handleInstanceReady(fake.instance));
		act(() =>
			fake.zrFire('mousemove', { event: { clientX: 10, clientY: 20 } }),
		);
		expect(result.current.mousePos).toEqual({ clientX: 10, clientY: 20 });
		act(() => fake.fire('updateAxisPointer', {}));
		act(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'p' }));
		});
		act(() =>
			fake.zrFire('mousemove', { event: { clientX: 99, clientY: 99 } }),
		);
		expect(result.current.mousePos).toEqual({ clientX: 10, clientY: 20 });
	});

	it('globalout은 핀이 아닐 때만 dataIndex를 지운다', () => {
		const fake = makeFakeChart();
		const { result } = renderHook(() =>
			useEChartsHoverPin({ canPinTooltip: true, resolveIndex: () => 5 }),
		);
		act(() => result.current.handleInstanceReady(fake.instance));
		act(() => fake.fire('updateAxisPointer', {}));
		act(() => fake.zrFire('globalout'));
		expect(result.current.hover.dataIndex).toBeNull();
	});

	it('dismissTooltip은 hover를 초기화한다', () => {
		const fake = makeFakeChart();
		const { result } = renderHook(() =>
			useEChartsHoverPin({ canPinTooltip: true, resolveIndex: () => 5 }),
		);
		act(() => result.current.handleInstanceReady(fake.instance));
		act(() => fake.fire('updateAxisPointer', {}));
		act(() => result.current.dismissTooltip());
		expect(result.current.hover).toEqual({ dataIndex: null, pinned: false });
	});
});
