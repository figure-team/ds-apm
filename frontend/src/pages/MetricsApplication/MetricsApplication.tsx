import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import { Tabs, TabsProps } from 'antd';
import { QueryParams } from 'constants/query';
import DBCall from 'container/MetricsApplication/Tabs/DBCall';
import External from 'container/MetricsApplication/Tabs/External';
import Overview from 'container/MetricsApplication/Tabs/Overview';
import ResourceAttributesFilter from 'container/ResourceAttributesFilter';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';

import ApDexApplication from './ApDex/ApDexApplication';
import { MetricsApplicationTab } from './types';
import useMetricsApplicationTabKey from './useMetricsApplicationTabKey';

import './MetricsApplication.styles.scss';

function MetricsApplication(): JSX.Element {
	const { servicename: encodedServiceName } = useParams<{
		servicename: string;
	}>();

	const activeKey = useMetricsApplicationTabKey();
	const { t } = useTranslation(['services']);

	const urlQuery = useUrlQuery();
	const { safeNavigate } = useSafeNavigate();

	const items: TabsProps['items'] = [
		{
			label: t('services:tab_overview'),
			key: MetricsApplicationTab.OVER_METRICS,
			children: <Overview />,
		},
		{
			label: t('services:tab_db_call_metrics'),
			key: MetricsApplicationTab.DB_CALL_METRICS,
			children: <DBCall />,
		},
		{
			label: t('services:tab_external_metrics'),
			key: MetricsApplicationTab.EXTERNAL_METRICS,
			children: <External />,
		},
	];

	const onTabChange = (tab: string): void => {
		urlQuery.set(QueryParams.tab, tab);
		safeNavigate(`/services/${encodedServiceName}?${urlQuery.toString()}`);
	};

	return (
		<div className="metrics-application-container">
			<ResourceAttributesFilter />
			<ApDexApplication />
			<Tabs
				items={items}
				activeKey={activeKey}
				className="service-route-tab"
				destroyInactiveTabPane
				onChange={onTabChange}
			/>
		</div>
	);
}

export default MetricsApplication;
