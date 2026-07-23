import { themeColors } from 'constants/theme';
import { generateColor } from 'lib/uPlotLib/utils/generateColor';

import { buildDsapmTheme, getSeriesColor } from '../themes/dsapmTheme';

describe('getSeriesColor', () => {
	it('colorMapping이 있으면 그 색을 그대로 쓴다 (uPlot 경로와 동일 우선순위)', () => {
		expect(getSeriesColor('api-latency', { 'api-latency': '#123456' }, true)).toBe(
			'#123456',
		);
	});

	it('매핑이 없으면 generateColor 파이프라인과 동일한 색 (다크)', () => {
		expect(getSeriesColor('api-latency', {}, true)).toBe(
			generateColor('api-latency', themeColors.chartcolors),
		);
	});

	it('라이트 모드는 lightModeColor 팔레트를 쓴다', () => {
		expect(getSeriesColor('api-latency', {}, false)).toBe(
			generateColor('api-latency', themeColors.lightModeColor),
		);
	});
});

describe('buildDsapmTheme', () => {
	it('다크/라이트 축 색이 서로 다르다', () => {
		const dark = buildDsapmTheme(true) as { categoryAxis: { axisLabel: { color: string } } };
		const light = buildDsapmTheme(false) as { categoryAxis: { axisLabel: { color: string } } };
		expect(dark.categoryAxis.axisLabel.color).not.toBe(
			light.categoryAxis.axisLabel.color,
		);
	});
});
