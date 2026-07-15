import { useMemo } from 'react';
import groupBy from 'lodash-es/groupBy';
import { useTranslation } from 'react-i18next';
import { Alerts } from 'types/api/alerts/getTriggered';

import { Value } from '../Filter';
import { FilterAlerts } from '../utils';
import { Container, TableHeader, TableHeaderContainer } from './styles';
import TableRowComponent from './TableRow';

function FilteredTable({
	selectedGroup,
	allAlerts,
	selectedFilter,
}: FilteredTableProps): JSX.Element {
	const { t } = useTranslation('alerts');

	const allGroupsAlerts = useMemo(
		() =>
			groupBy(FilterAlerts(allAlerts, selectedFilter), (obj) =>
				selectedGroup.map((e) => obj.labels?.[`${e.value}`]).join('+'),
			),
		[selectedGroup, allAlerts, selectedFilter],
	);

	const tags = Object.keys(allGroupsAlerts);
	const tagsAlerts = Object.values(allGroupsAlerts);

	const headers = [
		t('column_status'),
		t('column_alert_name'),
		t('column_severity'),
		t('triggered_column_firing_since'),
		t('triggered_column_tags'),
		// 'Actions',
	];

	return (
		<Container>
			<TableHeaderContainer>
				{headers.map((header) => (
					<TableHeader key={header} minWidth="90px">
						{header}
					</TableHeader>
				))}
			</TableHeaderContainer>

			{tags.map((e, index) => {
				const tagsValue = e.split('+').filter((e) => e);
				const tagsAlert: Alerts[] = tagsAlerts[index];

				if (tagsAlert.length === 0) {
					return null;
				}

				const { labels = {} } = tagsAlert[0];
				const keysArray = Object.keys(labels);
				const valueArray: string[] = [];

				keysArray.forEach((e) => {
					valueArray.push(labels[e]);
				});

				const tags = tagsValue
					.map((e) => keysArray[valueArray.findIndex((value) => value === e) || 0])
					.map((e, index) => `${e}:${tagsValue[index]}`);

				return <TableRowComponent key={e} tagsAlert={tagsAlert} tags={tags} />;
			})}
		</Container>
	);
}

interface FilteredTableProps {
	selectedGroup: Value[];
	allAlerts: Alerts[];
	selectedFilter: Value[];
}

export default FilteredTable;
