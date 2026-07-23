import { renderHook } from '@testing-library/react';

import { useEChartsEvents } from '../hooks/useEChartsEvents';

type Handler = (e: {
	offsetX: number;
	offsetY: number;
	event?: MouseEvent;
}) => void;

function makeFakeChart(): {
	chart: never;
	fire: (
		type: string,
		x: number,
		y: number,
		native?: MouseEvent,
	) => void;
	zrOff: jest.Mock;
	setOption: jest.Mock;
} {
	const handlers: Record<string, Handler> = {};
	const zrOff = jest.fn();
	const zr = {
		on: (type: string, h: Handler): void => {
			handlers[type] = h;
		},
		off: zrOff,
	};
	const setOption = jest.fn();
	const chart = {
		getZr: () => zr,
		// x 픽셀 → 시간(ms): 1px = 1000ms 선형 매핑으로 단순화
		convertFromPixel: (_: unknown, pixel: [number, number]) => [
			pixel[0] * 1000,
			0,
		],
		containPixel: () => true,
		setOption,
		getHeight: () => 200,
	};
	return {
		chart: chart as never,
		// native: 네이티브 MouseEvent(클릭 시 absoluteMouseX/Y 추출 경로 검증용). 미지정 시 undefined 유지
		fire: (type, x, y, native): void =>
			handlers[type]?.({ offsetX: x, offsetY: y, event: native }),
		zrOff,
		setOption,
	};
}

describe('useEChartsEvents', () => {
	it('10px 이상 드래그는 onDragSelect(startMs, endMs) 호출', () => {
		const { chart, fire } = makeFakeChart();
		const onDragSelect = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect }));

		fire('mousedown', 100, 50);
		fire('mousemove', 130, 50);
		fire('mouseup', 130, 50);

		expect(onDragSelect).toHaveBeenCalledWith(100000, 130000);
	});

	it('10px 미만 이동은 클릭 취급 (onDragSelect 미호출, onClick 호출)', () => {
		const { chart, fire } = makeFakeChart();
		const onDragSelect = jest.fn();
		const onClick = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect, onClick }));

		fire('mousedown', 100, 50);
		fire('mouseup', 105, 50); // 5px < 10px

		expect(onDragSelect).not.toHaveBeenCalled();
		expect(onClick).toHaveBeenCalledWith(
			expect.objectContaining({ mouseX: 105, mouseY: 50, xValueMs: 105000 }),
		);
	});

	it('역방향 드래그도 start < end로 정규화', () => {
		const { chart, fire } = makeFakeChart();
		const onDragSelect = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect }));

		fire('mousedown', 200, 50);
		fire('mouseup', 150, 50);

		expect(onDragSelect).toHaveBeenCalledWith(150000, 200000);
	});

	it('드래그 중 선택 사각형을 setOption으로 표시하고 mouseup 시 제거한다', () => {
		const { chart, fire, setOption } = makeFakeChart();
		const onDragSelect = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect }));

		fire('mousedown', 100, 50);
		fire('mousemove', 130, 50);

		// 드래그 중: graphic rect가 merge로 표시됨
		expect(setOption).toHaveBeenCalledWith(
			expect.objectContaining({
				graphic: [
					expect.objectContaining({
						id: 'dsapm-drag-selection',
						$action: 'merge',
					}),
				],
			}),
		);

		fire('mouseup', 130, 50);

		// 종료 시: graphic rect가 remove로 정리됨
		expect(setOption).toHaveBeenCalledWith(
			expect.objectContaining({
				graphic: [
					expect.objectContaining({
						id: 'dsapm-drag-selection',
						$action: 'remove',
					}),
				],
			}),
		);
	});

	it('클릭 시 네이티브 event.clientX/Y가 absoluteMouseX/Y로 전달된다', () => {
		const { chart, fire } = makeFakeChart();
		const onDragSelect = jest.fn();
		const onClick = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect, onClick }));

		fire('mousedown', 100, 50);
		fire('mouseup', 105, 50, { clientX: 505, clientY: 350 } as MouseEvent); // 5px < 10px, 네이티브 좌표 동반

		expect(onDragSelect).not.toHaveBeenCalled();
		expect(onClick).toHaveBeenCalledWith(
			expect.objectContaining({ absoluteMouseX: 505, absoluteMouseY: 350 }),
		);
	});

	it('정확히 10px 이동은 클릭이 아니라 드래그로 취급된다 (경계값)', () => {
		const { chart, fire } = makeFakeChart();
		const onDragSelect = jest.fn();
		const onClick = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect, onClick }));

		fire('mousedown', 100, 50);
		fire('mouseup', 110, 50); // 정확히 10px

		expect(onClick).not.toHaveBeenCalled();
		expect(onDragSelect).toHaveBeenCalledWith(100000, 110000);
	});

	it('mousemove 없이 클릭(mousedown→mouseup)만 발생하면 setOption을 전혀 호출하지 않는다', () => {
		const { chart, fire, setOption } = makeFakeChart();
		const onDragSelect = jest.fn();
		const onClick = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect, onClick }));

		fire('mousedown', 100, 50);
		fire('mouseup', 100, 50);

		expect(setOption).not.toHaveBeenCalled();
	});

	it('mousemove가 있었지만 10px 미만이라 사각형이 그려지지 않았다면 mouseup에서도 setOption을 호출하지 않는다', () => {
		const { chart, fire, setOption } = makeFakeChart();
		const onDragSelect = jest.fn();
		const onClick = jest.fn();
		renderHook(() => useEChartsEvents({ chart, onDragSelect, onClick }));

		fire('mousedown', 100, 50);
		fire('mousemove', 105, 50); // 5px < 10px, 사각형 미생성
		fire('mouseup', 105, 50);

		expect(setOption).not.toHaveBeenCalled();
	});

	it('언마운트 시 zr.off를 mousedown/mousemove/mouseup 3종에 대해 호출한다', () => {
		const { chart, zrOff } = makeFakeChart();
		const onDragSelect = jest.fn();
		const { unmount } = renderHook(() =>
			useEChartsEvents({ chart, onDragSelect }),
		);

		unmount();

		const offEventNames = zrOff.mock.calls.map((call) => call[0]);
		expect(offEventNames).toEqual(
			expect.arrayContaining(['mousedown', 'mousemove', 'mouseup']),
		);
		expect(zrOff).toHaveBeenCalledTimes(3);
	});
});
