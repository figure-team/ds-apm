// TooltipPlugin의 portal+style 배치 계층(TooltipPlugin.tsx:55-104) 대응.
// Tooltip 컴포넌트(Tooltip.tsx/TimeSeriesTooltip.tsx)는 스스로 위치를 잡지 않으므로
// 이 래퍼가 커서를 추적해 배치한다.
import { CSSProperties, ReactNode, useRef } from 'react';
import { createPortal } from 'react-dom';

const OFFSET_PX = 10;

interface Props {
	/** 네이티브 clientX/Y — EChartsTimeSeries가 zr mousemove로 갱신 */
	position: { clientX: number; clientY: number } | null;
	isPinned: boolean;
	children: ReactNode;
}

export default function EChartsTooltipPositioner({
	position,
	isPinned,
	children,
}: Props): JSX.Element | null {
	// 핀 시 마지막 위치 고정 — 비핀 상태에서만 최신 좌표로 갱신한다
	const pinnedPosRef = useRef(position);
	if (!isPinned) {
		pinnedPosRef.current = position;
	}
	const pos = isPinned ? pinnedPosRef.current : position;
	if (!pos) {
		return null;
	}

	// 뷰포트 클램프: 우/하단 넘침 시 반전. 실제 크기 측정이 필요하면
	// ResizeObserver 대신 max-width/height + transform 반전으로 단순화
	const flipX = pos.clientX > window.innerWidth * 0.6;
	const flipY = pos.clientY > window.innerHeight * 0.6;
	const style: CSSProperties = {
		position: 'fixed',
		left: pos.clientX + (flipX ? -OFFSET_PX : OFFSET_PX),
		top: pos.clientY + (flipY ? -OFFSET_PX : OFFSET_PX),
		transform: `translate(${flipX ? '-100%' : '0'}, ${flipY ? '-100%' : '0'})`,
		zIndex: 1000,
		// 핀 시에만 클릭 가능(핀 해제 버튼) — 비핀은 차트 클릭을 가로막지 않도록 통과
		pointerEvents: isPinned ? 'auto' : 'none',
	};
	return createPortal(
		<div style={style}>{children}</div>,
		(document.fullscreenElement as HTMLElement) ?? document.body,
	);
}
