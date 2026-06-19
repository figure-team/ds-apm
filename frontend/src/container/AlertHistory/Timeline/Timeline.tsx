import { useTranslation } from 'react-i18next';

import GraphWrapper from './GraphWrapper/GraphWrapper';
import TimelineTable from './Table/Table';
import TabsAndFilters from './TabsAndFilters/TabsAndFilters';

import './Timeline.styles.scss';

function TimelineTableRenderer(): JSX.Element {
	return <TimelineTable />;
}

function Timeline({
	totalCurrentTriggers,
}: {
	totalCurrentTriggers: number;
}): JSX.Element {
	const { t } = useTranslation('alerts');
	return (
		<div className="timeline">
			<div className="timeline__title">{t('hist_timeline')}</div>
			<div className="timeline__tabs-and-filters">
				<TabsAndFilters />
			</div>
			<div className="timeline__graph">
				<GraphWrapper totalCurrentTriggers={totalCurrentTriggers} />
			</div>
			<div className="timeline__table">
				<TimelineTableRenderer />
			</div>
		</div>
	);
}

export default Timeline;
