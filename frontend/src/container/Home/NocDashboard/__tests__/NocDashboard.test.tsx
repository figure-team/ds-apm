import { render, screen } from '@testing-library/react';

import NocDashboard from '../NocDashboard';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useDarkMode', () => ({ useIsDarkMode: () => true }));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate: jest.fn() }),
}));
jest.mock('container/TopNav/DateTimeSelectionV2', () => (): JSX.Element => (
	<div data-testid="dt" />
));
jest.mock('../components/InfraBadge', () => (): JSX.Element => (
	<div data-testid="infra-badge" />
));
jest.mock('../components/PinPickerDrawer', () => (): null => null);

const mockServices: unknown[] = [];
jest.mock('../hooks/useNocAlerts', () => ({
	__esModule: true,
	default: () => ({
		alerts: [],
		firingCount: 0,
		totalCount: 0,
		isLoading: false,
		isError: false,
	}),
}));
jest.mock('../hooks/useNocOverview', () => ({
	__esModule: true,
	default: () => ({
		services: mockServices,
		kpis: [],
		isLoading: false,
		isError: false,
	}),
}));
jest.mock('../hooks/useNocTrend', () => ({
	__esModule: true,
	default: () => ({ series: [], stepSec: 60, isLoading: false, isError: false }),
}));
jest.mock('../hooks/useNocInfra', () => ({
	__esModule: true,
	default: () => ({ hosts: [], isLoading: false, isError: false }),
}));
jest.mock('../hooks/useNocPinnedPanels', () => ({
	__esModule: true,
	default: () => ({
		slots: [],
		refs: [],
		dashboards: [],
		pin: jest.fn(),
		unpin: jest.fn(),
		isLoading: false,
	}),
}));

describe('NocDashboard (v4)', () => {
	beforeEach(() => {
		mockServices.length = 0;
	});

	it('renders single column: band badges + trend, without removed v2 sections', () => {
		render(<NocDashboard />);
		expect(screen.getByText('noc_c2_stable_title')).toBeInTheDocument();
		expect(screen.getByTestId('infra-badge')).toBeInTheDocument();
		// v4 제거 대상 부재 확인 — 우열 패널·OkStrip·평시 관찰대상
		expect(screen.queryByText('noc_c2_alerts_title')).toBeNull();
		expect(screen.queryByText('noc_c2_infra_title')).toBeNull();
		expect(screen.queryByText('noc_c2_ok_label')).toBeNull();
		expect(screen.queryByText('noc_c2_watch_normal')).toBeNull();
		// 평시엔 이상 서비스 배지도 없음
		expect(screen.queryByText('noc_c2_anom_badge')).toBeNull();
	});

	it('shows anomaly badge (not inline cards) when a service is unhealthy', () => {
		mockServices.push({
			name: 'payment-api',
			health: 'critical',
			p99Ms: 400,
			errPct: 12,
			rps: 10,
		});
		render(<NocDashboard />);
		expect(
			screen.getByRole('button', { name: /noc_c2_anom_badge/ }),
		).toBeInTheDocument();
		// 트렌드를 밀어내는 인라인 카드는 렌더되지 않는다 (말풍선 안에만 존재)
		expect(screen.queryByText('noc_c2_watch_anomaly')).toBeNull();
	});
});
