// ECharts 조립 공통 hover/핀 상태기계 (라인·막대·히스토그램 공유).
// 2a EChartsCartesian에 인라인이던 블록을 추출한 것으로 동작은 동일하다.
// 차트별로 다른 것은 "axisPointer 페이로드 → dataIndex" 해석뿐이라 주입받는다.
import { useCallback, useEffect, useRef, useState } from 'react';

import { EChartsType } from '../echartsCore';

// uPlot 경로 패리티: 핀은 호버 중 'p' 키 (DEFAULT_PIN_TOOLTIP_KEY — 클릭은 메뉴 전용)
const PIN_TOOLTIP_KEY = 'p';

export interface HoverState {
	dataIndex: number | null;
	pinned: boolean;
}

/** echarts 'updateAxisPointer' 이벤트 페이로드 — axisTrigger.js의 outputPayload 형태 */
export interface AxisPointerEventInfo {
	dataIndex?: number;
	axesInfo?: Array<{ axisDim: string; value?: number }>;
}

export interface UseEChartsHoverPinArgs {
	canPinTooltip: boolean;
	/** axisPointer 페이로드 → dataIndex. 라인/막대=최근접 타임스탬프, 히스토그램=버킷 bracket */
	resolveIndex: (info: AxisPointerEventInfo) => number | null;
}

export interface UseEChartsHoverPinResult {
	chart: EChartsType | null;
	hover: HoverState;
	mousePos: { clientX: number; clientY: number } | null;
	dismissTooltip: () => void;
	handleInstanceReady: (instance: EChartsType) => void;
}

export function useEChartsHoverPin({
	canPinTooltip,
	resolveIndex,
}: UseEChartsHoverPinArgs): UseEChartsHoverPinResult {
	const [chart, setChart] = useState<EChartsType | null>(null);
	const [hover, setHover] = useState<HoverState>({
		dataIndex: null,
		pinned: false,
	});
	const [mousePos, setMousePos] = useState<{
		clientX: number;
		clientY: number;
	} | null>(null);

	// handleInstanceReady는 인스턴스 생성 시 한 번만 호출되므로 그 안의 리스너가
	// 최신 resolveIndex(호출부가 매 렌더 새로 만드는 클로저)를 읽으려면 ref가 필요하다
	const resolveIndexRef = useRef(resolveIndex);
	resolveIndexRef.current = resolveIndex;

	// mousemove는 고빈도라 매 이벤트 setMousePos 시 리렌더가 잦다. rAF로 프레임당
	// 1회만 반영하고, 핀 상태에선 Positioner가 좌표를 무시하므로 갱신 자체를 건너뛴다.
	const pinnedRef = useRef(false);
	pinnedRef.current = hover.pinned;
	const mouseRafRef = useRef<number | null>(null);
	const pendingMouseRef = useRef<{ clientX: number; clientY: number } | null>(
		null,
	);

	const dismissTooltip = useCallback(
		(): void => setHover({ dataIndex: null, pinned: false }),
		[],
	);

	// 핀은 uPlot 경로 패리티대로 호버 중 'p' 키. Esc는 해제 전용.
	useEffect(() => {
		if (!canPinTooltip) {
			return undefined;
		}
		const onKeyDown = (e: KeyboardEvent): void => {
			if (e.key === 'Escape') {
				setHover((prev) => (prev.pinned ? { ...prev, pinned: false } : prev));
				return;
			}
			// uPlot 경로 패리티(TooltipPlugin.tsx:310) — 대소문자 무시 비교
			if (e.key.toLowerCase() !== PIN_TOOLTIP_KEY) {
				return;
			}
			setHover((prev) =>
				prev.dataIndex === null && !prev.pinned
					? prev
					: { ...prev, pinned: !prev.pinned },
			);
		};
		window.addEventListener('keydown', onKeyDown);
		return (): void => window.removeEventListener('keydown', onKeyDown);
	}, [canPinTooltip]);

	// 툴팁 상태: axisPointer 이벤트 → dataIndex 추적 (핀 상태에선 갱신 정지)
	const handleInstanceReady = useCallback((instance: EChartsType): void => {
		setChart(instance);
		instance.on('updateAxisPointer', (e: unknown): void => {
			const dataIndex = resolveIndexRef.current(e as AxisPointerEventInfo);
			setHover((prev) => (prev.pinned ? prev : { ...prev, dataIndex }));
		});
		// 툴팁 배치용 마우스 추적 — rAF로 프레임당 1회만 반영(고빈도 리렌더 방지).
		instance.getZr().on('mousemove', (e: { event?: MouseEvent }): void => {
			const native = e.event;
			if (!native || pinnedRef.current) {
				return;
			}
			pendingMouseRef.current = {
				clientX: native.clientX,
				clientY: native.clientY,
			};
			if (mouseRafRef.current !== null) {
				return;
			}
			mouseRafRef.current = requestAnimationFrame(() => {
				mouseRafRef.current = null;
				if (pendingMouseRef.current) {
					setMousePos(pendingMouseRef.current);
				}
			});
		});
		instance.getZr().on('globalout', (): void => {
			setHover((prev) => (prev.pinned ? prev : { ...prev, dataIndex: null }));
		});
	}, []);

	// 언마운트 시 대기 중인 mousemove rAF 취소 (해제 후 setState 방지)
	useEffect(
		() => (): void => {
			if (mouseRafRef.current !== null) {
				cancelAnimationFrame(mouseRafRef.current);
				mouseRafRef.current = null;
			}
		},
		[],
	);

	return { chart, hover, mousePos, dismissTooltip, handleInstanceReady };
}
