// echarts 트리셰이킹 단일 진입점 — 여기 외 다른 파일에서 'echarts'를 직접 임포트하지 않는다
import { LineChart } from 'echarts/charts';
import {
	AxisPointerComponent,
	GraphicComponent,
	GridComponent,
	MarkLineComponent,
} from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';

// 범례 UI는 기존 React Legend가 담당하고 시리즈 show/hide는 option 재빌드로
// 처리하므로 echarts LegendComponent는 등록하지 않는다(번들 절감). 시리즈 강조는
// highlight/downplay(코어 액션)라 LegendComponent가 필요 없다.
echarts.use([
	LineChart,
	GridComponent,
	MarkLineComponent,
	AxisPointerComponent,
	GraphicComponent,
	CanvasRenderer,
]);

export type { EChartsType } from 'echarts/core';
export type { EChartsCoreOption as EChartsOption } from 'echarts/core';
export default echarts;
