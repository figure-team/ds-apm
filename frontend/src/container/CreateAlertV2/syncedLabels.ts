import {
	convertExpressionToFilters,
	convertFiltersToExpression,
	removeKeysFromExpression,
} from 'components/QueryBuilderV2/utils';
import { Labels } from 'types/api/alerts/def';
import { TagFilterItem } from 'types/api/queryBuilder/queryBuilderData';

/**
 * Resource attributes that are kept in sync (both directions) between the
 * metric query builder's filter expression and the alert rule's labels.
 *
 * `attributeKey` is the query attribute name (dotted, e.g. `service.name`).
 * `labelKey` is the alert label key (underscored, Prometheus-friendly).
 */
export const MANAGED_ATTRIBUTES: Array<{
	attributeKey: string;
	labelKey: string;
}> = [
	// Service uses the dotted key to match the app's recommended operational
	// label + alert payload convention (see operationalMetadata.ts).
	{ attributeKey: 'service.name', labelKey: 'service.name' },
	{ attributeKey: 'deployment.environment', labelKey: 'deployment_environment' },
	{ attributeKey: 'host.name', labelKey: 'host_name' },
	{ attributeKey: 'k8s.namespace.name', labelKey: 'k8s_namespace_name' },
	{ attributeKey: 'k8s.cluster.name', labelKey: 'k8s_cluster_name' },
];

const attributeToLabel = new Map(
	MANAGED_ATTRIBUTES.map((m) => [m.attributeKey, m.labelKey]),
);

/** Whether a label key is one of the auto-synced resource attributes. */
export const isManagedLabelKey = (labelKey: string): boolean =>
	MANAGED_ATTRIBUTES.some((m) => m.labelKey === labelKey);

/**
 * Returns the single string value of a filter item when it is a simple
 * equality (`=`) or single-valued `IN`. Returns null for anything else
 * (negation, regex, multi-value `IN`, empty value, ...).
 */
const getSyncableValue = (item: TagFilterItem): string | null => {
	const op = (item.op || '').trim().toLowerCase();
	if (op !== '=' && op !== 'in') {
		return null;
	}

	let { value } = item;
	if (Array.isArray(value)) {
		if (value.length !== 1) {
			return null;
		}
		[value] = value;
	}

	if (value === '' || value === null || value === undefined) {
		return null;
	}

	return String(value);
};

/**
 * Extracts the managed resource attributes from a query filter expression and
 * returns them as alert labels (underscored keys).
 */
export const extractManagedLabels = (expression: string): Labels => {
	const labels: Labels = {};
	if (!expression) {
		return labels;
	}

	const items = convertExpressionToFilters(expression);
	items.forEach((item) => {
		const attributeKey = item.key?.key;
		if (!attributeKey) {
			return;
		}
		const labelKey = attributeToLabel.get(attributeKey);
		if (!labelKey) {
			return;
		}
		const value = getSyncableValue(item);
		if (value === null) {
			return;
		}
		labels[labelKey] = value;
	});

	return labels;
};

/**
 * Merges the managed labels derived from a query into an existing set of alert
 * labels. Managed keys present in the query are set, managed keys absent from
 * the query are removed; all non-managed labels (including `severity`) are kept
 * untouched.
 */
export const mergeManagedLabels = (
	existing: Labels | undefined,
	fromQuery: Labels,
): Labels => {
	const result: Labels = { ...(existing || {}) };
	MANAGED_ATTRIBUTES.forEach(({ labelKey }) => {
		if (fromQuery[labelKey] !== undefined) {
			result[labelKey] = fromQuery[labelKey];
		} else {
			delete result[labelKey];
		}
	});
	return result;
};

/**
 * Applies changes to managed labels back onto a query filter expression.
 *
 * Only the managed attributes whose value actually changed (compared to what
 * the expression already encodes) are rewritten, so unrelated clauses — and
 * non-syncable shapes such as multi-value `IN` for the same key — are left
 * untouched unless the user explicitly set a new single value.
 *
 * Returns the (possibly unchanged) expression string.
 */
export const syncLabelsToExpression = (
	expression: string,
	labels: Labels,
): string => {
	const current = extractManagedLabels(expression || '');

	const changedAttributeKeys: string[] = [];
	const itemsToAdd: TagFilterItem[] = [];

	MANAGED_ATTRIBUTES.forEach(({ attributeKey, labelKey }) => {
		const rawNew = labels[labelKey];
		const normalizedNew = rawNew === '' ? undefined : rawNew;
		const oldValue = current[labelKey];

		if (normalizedNew === oldValue) {
			return;
		}

		changedAttributeKeys.push(attributeKey);
		if (normalizedNew !== undefined) {
			itemsToAdd.push({
				id: '',
				key: { id: attributeKey, key: attributeKey, type: '' },
				op: '=',
				value: normalizedNew,
			});
		}
	});

	if (changedAttributeKeys.length === 0) {
		return expression || '';
	}

	let updated = removeKeysFromExpression(expression || '', changedAttributeKeys);

	if (itemsToAdd.length > 0) {
		const { expression: addition } = convertFiltersToExpression({
			items: itemsToAdd,
			op: 'AND',
		});
		if (addition) {
			updated = updated ? `${updated} AND ${addition}` : addition;
		}
	}

	return updated;
};
