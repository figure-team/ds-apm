import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
// eslint-disable-next-line no-restricted-imports
import { useDispatch, useSelector } from 'react-redux';
import { useHistory, useLocation } from 'react-router-dom';
import { Alert, Typography } from 'antd';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import { LOCALSTORAGE } from 'constants/localStorage';
import { QueryParams } from 'constants/query';
import { PANEL_TYPES } from 'constants/queryBuilder';
import GridCard from 'container/GridCardLayout/GridCard';
import { Card, CardContainer } from 'container/GridCardLayout/styles';
import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';
import dayjs from 'dayjs';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import useUrlQuery from 'hooks/useUrlQuery';
import { UpdateTimeInterval } from 'store/actions';
import { AppState } from 'store/reducers';
import { Widgets } from 'types/api/dashboard/getAll';
import { GlobalReducer } from 'types/reducer/globalTime';
import { v4 as uuid } from 'uuid';

import {
	getLogCountWidgetData,
	getLogSizeWidgetData,
	getMetricCountWidgetData,
	getSpanCountWidgetData,
	getSpanSizeWidgetData,
	getTotalLogSizeWidgetData,
	getTotalMetricDatapointCountWidgetData,
	getTotalTraceSizeWidgetData,
} from './graphs';

import './BreakDown.styles.scss';

type MetricSection = {
	id: string;
	// i18n key in the `meter` namespace — translated at render time in Section
	title: string;
	graphs: Widgets[];
};


function Section(section: MetricSection): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { t } = useTranslation('meter');
	const { title, graphs } = section;
	const history = useHistory();
	const { pathname } = useLocation();
	const dispatch = useDispatch();
	const urlQuery = useUrlQuery();

	const onDragSelect = useCallback(
		(start: number, end: number) => {
			const startTimestamp = Math.trunc(start);
			const endTimestamp = Math.trunc(end);

			urlQuery.set(QueryParams.startTime, startTimestamp.toString());
			urlQuery.set(QueryParams.endTime, endTimestamp.toString());
			const generatedUrl = `${pathname}?${urlQuery.toString()}`;
			history.push(generatedUrl);

			if (startTimestamp !== endTimestamp) {
				dispatch(UpdateTimeInterval('custom', [startTimestamp, endTimestamp]));
			}
		},
		[dispatch, history, pathname, urlQuery],
	);

	return (
		<div className="meter-column-graph">
			<CardContainer className="row-card" isDarkMode={isDarkMode}>
				<Typography.Text className="section-title">{t(title)}</Typography.Text>
			</CardContainer>
			<div className="meter-page-grid">
				{graphs.map((widget) => (
					<Card
						key={widget?.id}
						isDarkMode={isDarkMode}
						$panelType={PANEL_TYPES.BAR}
						className="meter-graph"
					>
						<GridCard widget={widget} onDragSelect={onDragSelect} version="v5" />
					</Card>
				))}
			</div>
		</div>
	);
}

function BreakDown(): JSX.Element {
	const { t } = useTranslation(['meter', 'common']);

	const sections = useMemo<MetricSection[]>(
		() => [
			{
				id: uuid(),
				title: 'section_total',
				graphs: [
					getTotalLogSizeWidgetData(),
					getTotalTraceSizeWidgetData(),
					getTotalMetricDatapointCountWidgetData(),
				],
			},
			{
				id: uuid(),
				title: 'section_logs',
				graphs: [getLogCountWidgetData(), getLogSizeWidgetData()],
			},
			{
				id: uuid(),
				title: 'section_traces',
				graphs: [getSpanCountWidgetData(), getSpanSizeWidgetData()],
			},
			{
				id: uuid(),
				title: 'section_metrics',
				graphs: [getMetricCountWidgetData()],
			},
		],
		// t changes reference when meter namespace finishes loading → triggers retranslation
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[t],
	);

	const { isCloudUser } = useGetTenantLicense();
	const { maxTime, minTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);

	const showInfo =
		getLocalStorageApi(LOCALSTORAGE.DISSMISSED_COST_METER_INFO) !== 'true';
	const isDateBeforeAugust22nd2025 = (minTime: number): boolean => {
		const august22nd2025UTC = dayjs.utc('2025-08-22T00:00:00Z');
		return dayjs(minTime / 1e6).isBefore(august22nd2025UTC);
	};
	const showShortRangeWarning = (maxTime - minTime) / 1e6 < 61 * 60 * 1000;

	return (
		<div className="meter-explorer-breakdown">
			<section className="meter-explorer-date-time">
				<DateTimeSelectionV2 showAutoRefresh={false} />
			</section>
			<section className="meter-explorer-graphs">
				{showInfo && (
					<Alert
						type="info"
						showIcon
						closable
						onClose={(): void => {
							setLocalStorageApi(LOCALSTORAGE.DISSMISSED_COST_METER_INFO, 'true');
						}}
						message={t('billing_utc_info')}
					/>
				)}
				{isCloudUser && isDateBeforeAugust22nd2025(minTime) && (
					<Alert
						type="warning"
						showIcon
						message={t('accuracy_warning')}
					/>
				)}

				{showShortRangeWarning && (
					<Alert
						type="warning"
						showIcon
						closable
						message={
							<>
								{t('aggregation_warning')}&nbsp;
								<a
									href="https://signoz.io/docs/cost-meter/overview/#accessing-cost-meter"
									rel="noopener noreferrer"
									target="_blank"
									style={{ textDecoration: 'underline' }}
								>
									{t('common:learn_more')}
								</a>
								.
							</>
						}
					/>
				)}
				<section className="total">
					<Section
						id={sections[0].id}
						title={sections[0].title}
						graphs={sections[0].graphs}
					/>
				</section>
				{sections.map((section, idx) => {
					if (idx === 0) {
						return;
					}

					return (
						<Section
							key={section.id}
							id={section.id}
							title={section.title}
							graphs={section.graphs}
						/>
					);
				})}
			</section>
		</div>
	);
}

export default BreakDown;
