import { useEffect, useRef } from 'react';

import { EChartsType } from '../echartsCore';

// uPlot drag.dist(10px) 패리티 — 미세 드래그가 줌으로 오인되는 회귀(2242cbc4) 방지
export const DRAG_CLICK_DIST_PX = 10;

const SELECTION_RECT_ID = 'dsapm-drag-selection';

export interface EChartsClickInfo {
	mouseX: number;
	mouseY: number;
	absoluteMouseX: number;
	absoluteMouseY: number;
	xValueMs: number | null;
	yValue: number | null;
}

interface UseEChartsEventsArgs {
	chart: EChartsType | null;
	onDragSelect: (startTime: number, endTime: number) => void;
	onClick?: (click: EChartsClickInfo) => void;
}

export function useEChartsEvents({
	chart,
	onDragSelect,
	onClick,
}: UseEChartsEventsArgs): void {
	// 콜백 최신 참조 유지 (리스너 재등록 방지)
	const callbacksRef = useRef({ onDragSelect, onClick });
	callbacksRef.current = { onDragSelect, onClick };

	useEffect(() => {
		if (!chart) {
			return undefined;
		}
		const zr = chart.getZr();
		let dragStartX: number | null = null;
		let dragStartY = 0;
		// 선택 사각형이 실제로 merge로 그려졌는지 추적 (미그려진 상태에서 remove 시 echarts 5.6 TypeError 방지)
		let rectVisible = false;

		const pixelToValue = (
			x: number,
			y: number,
		): [number | null, number | null] => {
			const converted = chart.convertFromPixel({ gridIndex: 0 }, [x, y]);
			return Array.isArray(converted)
				? [converted[0] ?? null, converted[1] ?? null]
				: [null, null];
		};

		const clearSelectionRect = (): void => {
			chart.setOption({
				graphic: [{ id: SELECTION_RECT_ID, $action: 'remove' }],
			});
		};

		const onMouseDown = (e: { offsetX: number; offsetY: number }): void => {
			dragStartX = e.offsetX;
			dragStartY = e.offsetY;
		};

		const onMouseMove = (e: { offsetX: number; offsetY: number }): void => {
			if (dragStartX === null) {
				return;
			}
			if (Math.abs(e.offsetX - dragStartX) < DRAG_CLICK_DIST_PX) {
				return;
			}
			// 드래그 선택 영역 표시
			chart.setOption({
				graphic: [
					{
						id: SELECTION_RECT_ID,
						type: 'rect',
						$action: 'merge',
						silent: true,
						shape: {
							x: Math.min(dragStartX, e.offsetX),
							y: 0,
							width: Math.abs(e.offsetX - dragStartX),
							height: chart.getHeight(),
						},
						style: { fill: 'rgba(216, 27, 44, 0.08)' },
					},
				],
			});
			rectVisible = true;
		};

		const onMouseUp = (e: { offsetX: number; offsetY: number }): void => {
			if (dragStartX === null) {
				return;
			}
			const startX = dragStartX;
			dragStartX = null;
			if (rectVisible) {
				clearSelectionRect();
				rectVisible = false;
			}

			const dist = Math.abs(e.offsetX - startX);
			if (dist < DRAG_CLICK_DIST_PX) {
				// 10px 미만 = 클릭 (uPlot dist 패리티)
				const [xValueMs, yValue] = pixelToValue(e.offsetX, e.offsetY);
				const native = (e as { event?: MouseEvent }).event;
				callbacksRef.current.onClick?.({
					mouseX: e.offsetX,
					mouseY: e.offsetY,
					absoluteMouseX: native?.clientX ?? e.offsetX,
					absoluteMouseY: native?.clientY ?? e.offsetY,
					xValueMs,
					yValue,
				});
				return;
			}

			const [t1] = pixelToValue(startX, dragStartY);
			const [t2] = pixelToValue(e.offsetX, e.offsetY);
			if (t1 === null || t2 === null) {
				return;
			}
			callbacksRef.current.onDragSelect(Math.min(t1, t2), Math.max(t1, t2));
		};

		zr.on('mousedown', onMouseDown);
		zr.on('mousemove', onMouseMove);
		zr.on('mouseup', onMouseUp);
		return (): void => {
			zr.off('mousedown', onMouseDown);
			zr.off('mousemove', onMouseMove);
			zr.off('mouseup', onMouseUp);
		};
	}, [chart]);
}
