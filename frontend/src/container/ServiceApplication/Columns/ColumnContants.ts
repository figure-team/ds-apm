export enum ColumnKey {
	Application = 'serviceName',
	P99 = 'p99',
	ErrorRate = 'errorRate',
	Operations = 'callRate',
}

export const ColumnTitleKey: Record<ColumnKey, string> = {
	[ColumnKey.Application]: 'column_application',
	[ColumnKey.P99]: 'column_p99_latency',
	[ColumnKey.ErrorRate]: 'column_error_rate_pct',
	[ColumnKey.Operations]: 'column_operations_per_second',
};

export enum ColumnWidth {
	Application = 200,
	P99 = 150,
	ErrorRate = 150,
	Operations = 150,
}

export const SORTING_ORDER = 'descend';
