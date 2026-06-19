import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { Color } from '@signozhq/design-tokens';
import { TimelineFilter, TimelineTab } from 'container/AlertHistory/types';
import history from 'lib/history';
import { Info } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import Tabs2 from 'periscope/components/Tabs2';

import './TabsAndFilters.styles.scss';

function ComingSoon(): JSX.Element {
	const { t } = useTranslation('alerts');
	return (
		<div className="coming-soon">
			<div className="coming-soon__text">{t('hist_coming_soon')}</div>
			<div className="coming-soon__icon">
				<Info size={10} color={Color.BG_SIENNA_400} />
			</div>
		</div>
	);
}
function TimelineTabs(): JSX.Element {
	const { t } = useTranslation('alerts');
	const tabs = [
		{
			value: TimelineTab.OVERALL_STATUS,
			label: t('hist_overall_status'),
		},
		{
			value: TimelineTab.TOP_5_CONTRIBUTORS,
			label: (
				<div className="top-5-contributors">
					{t('hist_top5_contributors')}
					<ComingSoon />
				</div>
			),
			disabled: true,
		},
	];

	return <Tabs2 tabs={tabs} initialSelectedTab={TimelineTab.OVERALL_STATUS} />;
}

function TimelineFilters(): JSX.Element {
	const { t } = useTranslation('alerts');
	const { search } = useLocation();
	const searchParams = useMemo(() => new URLSearchParams(search), [search]);

	const initialSelectedTab = useMemo(
		() => searchParams.get('timelineFilter') ?? TimelineFilter.ALL,
		[searchParams],
	);

	const handleFilter = (value: TimelineFilter): void => {
		searchParams.set('timelineFilter', value);
		history.push({ search: searchParams.toString() });
	};

	const tabs = [
		{
			value: TimelineFilter.ALL,
			label: t('hist_tab_all'),
		},
		{
			value: TimelineFilter.FIRED,
			label: t('hist_tab_fired'),
		},
		{
			value: TimelineFilter.RESOLVED,
			label: t('hist_tab_resolved'),
		},
	];

	return (
		<Tabs2
			tabs={tabs}
			initialSelectedTab={initialSelectedTab}
			onSelectTab={handleFilter}
			hasResetButton
		/>
	);
}

function TabsAndFilters(): JSX.Element {
	return (
		<div className="timeline-tabs-and-filters">
			<TimelineTabs />
			<TimelineFilters />
		</div>
	);
}

export default TabsAndFilters;
