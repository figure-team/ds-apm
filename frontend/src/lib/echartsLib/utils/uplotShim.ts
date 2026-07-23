import type uPlot from 'uplot';

export interface ShimSeriesMeta {
	label: string;
	color: string;
	show: boolean;
}

/**
 * TimeSeriesTooltip(TooltipRenderArgs.uPlotInstance)이 접근하는
 * uPlot 표면만 흉내내는 심(shim).
 *
 * 조사 목록(Step 1 — 아래 파일을 직접 열어 확인한 실제 접근 표면):
 * ① data, series[i].{label, show, stroke(u, idx)}
 *    — TimeSeriesTooltip.tsx가 buildTooltipContent에 data/series를 그대로 전달하고,
 *      buildTooltipContent(Tooltip/utils.ts:56-109)가 seriesIndex 1..n을 순회하며
 *      series[i].show(불리언), series[i].label, resolveSeriesColor를 통해
 *      series[i].stroke(u, seriesIndex)를 호출한다.
 * ② cursor.idx + data[0][idx]
 *    — TooltipHeader.tsx:37-47이 uPlotInstance.cursor.idx로 커서 인덱스를 얻고
 *      uPlotInstance.data[0]?.[cursorIdx]로 타임스탬프(초 단위)를 읽어 헤더를 만든다.
 * ③ 배치 관련 접근 없음 — Tooltip.tsx/TooltipList.tsx/TooltipItem.tsx/TooltipFooter.tsx를
 *    모두 확인했으나 uPlotInstance의 다른 프로퍼티(posToVal 등)에 접근하는 코드는 없다.
 *    배치는 TooltipPlugin(uPlot 경로) 또는 EChartsTooltipPositioner(ECharts 경로) 소관.
 *
 * posToVal/valToPos/over는 확인된 접근 표면에는 없지만, 향후 TimeSeriesTooltip이
 * 확장되어 참조할 가능성에 대비한 방어적 안전망(항등/0 반환)이다.
 * 구현 중 추가 접근이 발견되면 여기에 추가한다.
 */
export function buildUPlotShim(
	chartData: uPlot.AlignedData,
	seriesMeta: ShimSeriesMeta[],
	cursorIdx: number | null,
): uPlot {
	const series = [
		{ label: 'Timestamp', show: true }, // x축 자리 (uPlot 규약)
		...seriesMeta.map((meta) => ({
			label: meta.label,
			show: meta.show,
			scale: 'y',
			stroke: (): string => meta.color,
		})),
	];

	return ({
		data: chartData,
		series,
		// 방어적 안전망 — 확인된 접근 표면에는 없음(주석 참조)
		posToVal: (pos: number): number => pos,
		valToPos: (): number => 0,
		over: typeof document !== 'undefined' ? document.createElement('div') : undefined,
		// 타임스탬프 헤더의 출처 (리뷰 반영 — hover dataIndex 주입)
		cursor: { idx: cursorIdx },
	} as unknown) as uPlot;
}
