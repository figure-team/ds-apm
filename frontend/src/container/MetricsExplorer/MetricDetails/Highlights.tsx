import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Button, Spin, Tooltip, Typography } from 'antd';
import { useGetMetricHighlights } from 'api/generated/services/metrics';
import { InfoIcon } from 'lucide-react';

import { formatNumberIntoHumanReadableFormat } from '../Summary/utils';
import { HighlightsProps } from './types';
import {
	formatNumberToCompactFormat,
	formatTimestampToReadableDate,
} from './utils';

function Highlights({ metricName }: HighlightsProps): JSX.Element {
	const { t } = useTranslation('metricsExplorer');
	const {
		data: metricHighlightsData,
		isLoading: isLoadingMetricHighlights,
		isError: isErrorMetricHighlights,
		refetch: refetchMetricHighlights,
	} = useGetMetricHighlights(
		{
			metricName,
		},
		{
			query: {
				enabled: !!metricName,
			},
		},
	);

	const metricHighlights = metricHighlightsData?.data;

	const timeSeriesActive = formatNumberToCompactFormat(
		metricHighlights?.activeTimeSeries,
	);
	const timeSeriesTotal = formatNumberToCompactFormat(
		metricHighlights?.totalTimeSeries,
	);
	const lastReceivedText = formatTimestampToReadableDate(
		metricHighlights?.lastReceived,
	);

	if (isErrorMetricHighlights) {
		return (
			<div className="metric-details-content-grid">
				<div
					className="metric-highlights-error-state"
					data-testid="metric-highlights-error-state"
				>
					<InfoIcon size={16} color={Color.BG_CHERRY_500} />
					<Typography.Text>
						{t('highlights_error')}
					</Typography.Text>
					<Button
						type="link"
						size="large"
						onClick={(): void => {
							refetchMetricHighlights();
						}}
					>
						{t('retry_q')}
					</Button>
				</div>
			</div>
		);
	}

	return (
		<div className="metric-details-content-grid">
			<div className="labels-row">
				<Typography.Text type="secondary" className="metric-details-grid-label">
					{t('samples')}
				</Typography.Text>
				<Typography.Text type="secondary" className="metric-details-grid-label">
					{t('time_series')}
				</Typography.Text>
				<Typography.Text type="secondary" className="metric-details-grid-label">
					{t('last_received')}
				</Typography.Text>
			</div>
			<div className="values-row">
				{isLoadingMetricHighlights ? (
					<div className="metric-highlights-loading-inline">
						<Spin size="small" />
						<Typography.Text type="secondary">{t('loading_metric_stats')}</Typography.Text>
					</div>
				) : (
					<>
						<Typography.Text
							className="metric-details-grid-value"
							data-testid="metric-highlights-data-points"
						>
							<Tooltip title={metricHighlights?.dataPoints?.toLocaleString()}>
								{formatNumberIntoHumanReadableFormat(metricHighlights?.dataPoints ?? 0)}
							</Tooltip>
						</Typography.Text>
						<Typography.Text
							className="metric-details-grid-value"
							data-testid="metric-highlights-time-series-total"
						>
							<Tooltip
								title={t('active_time_series_tooltip')}
								placement="top"
							>
								<span>{`${timeSeriesTotal} total ⎯ ${timeSeriesActive} active`}</span>
							</Tooltip>
						</Typography.Text>
						<Typography.Text
							className="metric-details-grid-value"
							data-testid="metric-highlights-last-received"
						>
							<Tooltip title={lastReceivedText}>{lastReceivedText}</Tooltip>
						</Typography.Text>
					</>
				)}
			</div>
		</div>
	);
}

export default Highlights;
