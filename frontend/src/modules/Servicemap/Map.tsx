/* eslint-disable  */
//@ts-nocheck
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { useIsDarkMode } from 'hooks/useDarkMode';

import { getGraphData, getTooltip, transformLabel } from './utils';

function ServiceMap({ fgRef, serviceMap }: any): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const wrapperRef = useRef(null);
	// width/height를 넘기지 않으면 라이브러리가 캔버스를 창 크기로 만들어
	// 컨테이너를 넘치고(우측 클리핑), 그래프가 뷰포트 밖에 렌더될 수 있다.
	const [size, setSize] = useState({ w: 0, h: 0 });
	const didFitRef = useRef(false);

	useEffect(() => {
		const el = wrapperRef.current;
		if (!el) {
			return undefined;
		}
		const measure = (): void => {
			const rect = el.getBoundingClientRect();
			setSize({
				w: rect.width,
				h: Math.max(400, window.innerHeight - rect.top - 16),
			});
		};
		measure();
		window.addEventListener('resize', measure);
		let ro;
		if (typeof ResizeObserver !== 'undefined') {
			ro = new ResizeObserver(measure);
			ro.observe(el);
		}
		return (): void => {
			window.removeEventListener('resize', measure);
			ro && ro.disconnect();
		};
	}, []);

	// 데이터가 바뀌면 시뮬레이션 종료 시점에 다시 화면에 맞춘다.
	useEffect(() => {
		didFitRef.current = false;
	}, [serviceMap]);

	const handleEngineStop = useCallback(() => {
		if (!didFitRef.current && fgRef.current) {
			fgRef.current.zoomToFit(400, 60);
			didFitRef.current = true;
		}
	}, [fgRef]);

	const { nodes, links } = getGraphData(serviceMap, isDarkMode);

	const graphData = { nodes, links };

	let zoomLevel = 1;

	return (
		<div ref={wrapperRef} style={{ width: '100%' }}>
			{size.w > 0 && (
				<ForceGraph2D
					ref={fgRef}
					width={size.w}
					height={size.h}
					cooldownTicks={100}
					onEngineStop={handleEngineStop}
					graphData={graphData}
					linkLabel={getTooltip}
					linkAutoColorBy={(d) => d.target}
					linkDirectionalParticles="value"
					linkDirectionalParticleSpeed={(d) => d.value}
					nodeCanvasObject={(node, ctx) => {
						const label = transformLabel(node.id, zoomLevel);
						let { fontSize } = node;
						fontSize = (fontSize * 3) / zoomLevel;
						ctx.font = `${fontSize}px Roboto`;
						const { width } = node;

						ctx.fillStyle = node.color;
						ctx.beginPath();
						ctx.arc(node.x, node.y, width, 0, 2 * Math.PI, false);
						ctx.fill();
						ctx.textAlign = 'center';
						ctx.textBaseline = 'middle';
						ctx.fillStyle = isDarkMode ? '#ffffff' : '#000000';
						ctx.fillText(label, node.x, node.y);
					}}
					onLinkHover={(node) => {
						const tooltip = document.querySelector('.graph-tooltip');
						if (tooltip && node) {
							tooltip.innerHTML = getTooltip(node);
						}
					}}
					onZoom={(zoom) => {
						zoomLevel = zoom.k;
					}}
					nodePointerAreaPaint={(node, color, ctx) => {
						ctx.fillStyle = color;
						ctx.beginPath();
						ctx.arc(node.x, node.y, 5, 0, 2 * Math.PI, false);
						ctx.fill();
					}}
				/>
			)}
		</div>
	);
}

export default memo(ServiceMap);
