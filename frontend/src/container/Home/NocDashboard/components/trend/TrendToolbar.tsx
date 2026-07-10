import { useTranslation } from 'react-i18next';

import { TrendMetric } from '../../types';

const METRICS: TrendMetric[] = ['err', 'p99', 'rps'];

export interface TrendToolbarProps {
	metric: TrendMetric;
	onMetricChange: (m: TrendMetric) => void;
	logScale: boolean;
	onLogScaleChange: (v: boolean) => void;
}

export default function TrendToolbar({
	metric,
	onMetricChange,
	logScale,
	onLogScaleChange,
}: TrendToolbarProps): JSX.Element {
	const { t } = useTranslation('home');
	return (
		<div className="noc-c2-trend-tabs">
			{METRICS.map((m) => (
				<button
					key={m}
					type="button"
					className={m === metric ? 'active' : ''}
					onClick={(): void => onMetricChange(m)}
				>
					{t(`noc_c2_metric_${m}`)}
				</button>
			))}
			{/* 오류율(0~100%)은 선형이 자연스러워 로그 토글을 p99/rps에서만 노출 */}
			{metric !== 'err' ? (
				<button
					type="button"
					className={`noc-c2-log-toggle${logScale ? ' active' : ''}`}
					onClick={(): void => onLogScaleChange(!logScale)}
				>
					{t('noc_c2_log_scale')}
				</button>
			) : null}
		</div>
	);
}
