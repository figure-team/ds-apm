// echarts 트리셰이킹 단일 진입점 — 여기 외 다른 파일에서 'echarts'를 직접 임포트하지 않는다
import { LineChart } from 'echarts/charts';
import {
	AxisPointerComponent,
	GraphicComponent,
	GridComponent,
	// 범례 UI는 기존 React Legend를 쓴다. 시리즈 show/hide는 option 재빌드
	// 방식(리뷰 반영)이라 legend 액션은 쓰지 않지만, option에 legend(show:false)
	// 키가 있으므로 컴포넌트 등록은 유지한다
	LegendComponent,
	MarkLineComponent,
} from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';

echarts.use([
	LineChart,
	GridComponent,
	MarkLineComponent,
	LegendComponent,
	AxisPointerComponent,
	GraphicComponent,
	CanvasRenderer,
]);

export type { EChartsType } from 'echarts/core';
export type { EChartsCoreOption as EChartsOption } from 'echarts/core';
export default echarts;
