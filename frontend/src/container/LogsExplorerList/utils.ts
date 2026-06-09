import { TelemetryFieldKey } from 'api/v5/v5';
import { isEmpty } from 'lodash-es';
import { TFunction } from 'i18next';
import { IField } from 'types/api/logs/fields';
import {
	IBuilderQuery,
	TagFilterItem,
} from 'types/api/queryBuilder/queryBuilderData';

export const convertKeysToColumnFields = (
	keys: TelemetryFieldKey[],
): IField[] =>
	keys
		.filter((item) => !isEmpty(item.name))
		.map((item) => ({
			dataType: item.fieldDataType ?? '',
			name: item.name,
			type: item.fieldContext ?? '',
		}));
/**
 * Determines if a query represents a trace-to-logs navigation
 * by checking for the presence of a trace_id filter.
 */
export const isTraceToLogsQuery = (queryData: IBuilderQuery): boolean => {
	// Check if this is a trace-to-logs query by looking for trace_id filter
	if (!queryData?.filters?.items) {
		return false;
	}

	const traceIdFilter = queryData.filters.items.find(
		(item: TagFilterItem) => item.key?.key === 'trace_id',
	);

	return !!traceIdFilter;
};

export type EmptyLogsListConfig = {
	title: string;
	subTitle: string;
	description: string | string[];
	documentationLinks?: Array<{
		text: string;
		url: string;
	}>;
	showClearFiltersButton?: boolean;
	onClearFilters?: () => void;
	clearFiltersButtonText?: string;
};

export const getEmptyLogsListConfig = (
	handleClearFilters: () => void,
	t: TFunction,
): EmptyLogsListConfig => ({
	title: t('logs:no_logs_found_for_trace'),
	subTitle: t('logs:this_could_be_because'),
	description: [
		t('logs:logs_not_linked_to_traces'),
		t('logs:logs_not_being_sent'),
		t('logs:no_logs_associated_with_trace'),
	],
	documentationLinks: [
		{
			text: t('logs:sending_logs_to_signoz'),
			url: 'https://signoz.io/docs/logs-management/send-logs-to-signoz/',
		},
		{
			text: t('logs:correlate_traces_and_logs'),
			url: 'https://signoz.io/docs/traces-management/guides/correlate-traces-and-logs/',
		},
	],
	clearFiltersButtonText: t('logs:clear_filters_from_trace'),
	showClearFiltersButton: true,
	onClearFilters: handleClearFilters,
});
