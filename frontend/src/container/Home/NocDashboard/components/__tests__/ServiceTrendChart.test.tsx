import { render, screen } from '@testing-library/react';

import { TrendSeries } from '../../types';
import { computeScale } from '../../utils/trendScale';
import ServiceTrendChart from '../../ServiceTrendChart';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useDarkMode', () => ({ useIsDarkMode: () => true }));

const series: TrendSeries[] = [
	{ name: 'cart', color: '#3987e5', points: [{ t: 1000, v: 5 }, { t: 2000, v: 15 }] },
	{
		name: 'auth',
		color: '#199e70',
		points: [{ t: 1000, v: 0 }, { t: 2000, v: 8 }],
		missing: false,
	},
	{ name: 'gone', color: '#c98500', points: [], missing: true },
];

describe('computeScale', () => {
	it('derives min/max across non-missing series with headroom', () => {
		const s = computeScale(series, 'err');
		expect(s.maxV).toBeGreaterThanOrEqual(15);
		expect(s.minV).toBeLessThanOrEqual(0);
		expect(s.minT).toBe(1000);
		expect(s.maxT).toBe(2000);
	});

	it('empty (all missing) yields safe defaults, no NaN', () => {
		const s = computeScale(
			[{ name: 'x', color: '#000', points: [], missing: true }],
			'rps',
		);
		expect(Number.isFinite(s.maxV)).toBe(true);
		expect(Number.isFinite(s.minV)).toBe(true);
	});
});

describe('ServiceTrendChart', () => {
	it('renders metric toggle and legend with missing marker', () => {
		render(
			<ServiceTrendChart
				series={series}
				metric="err"
				onMetricChange={jest.fn()}
				loading={false}
				error={false}
			/>,
		);
		expect(screen.getByText('noc_c2_metric_err')).toBeInTheDocument();
		expect(screen.getByText('cart')).toBeInTheDocument();
		expect(screen.getByText('noc_c2_series_nodata')).toBeInTheDocument(); // missing legend
	});

	it('shows error state when error=true', () => {
		render(
			<ServiceTrendChart
				series={[]}
				metric="err"
				onMetricChange={jest.fn()}
				loading={false}
				error
			/>,
		);
		expect(screen.getByText('noc_c2_trend_error')).toBeInTheDocument();
	});
});
