import { ENTITY_VERSION_V5 } from 'constants/app';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { useGetExplorerQueryRange } from 'hooks/queryBuilder/useGetExplorerQueryRange';
import cloneDeep from 'lodash-es/cloneDeep';
import { useMemo } from 'react';
import { DataSource } from 'types/common/queryBuilder';

import { NocLogLine } from '../types';

const LEVELS: NocLogLine['level'][] = ['ERROR', 'WARN', 'INFO', 'DEBUG'];

function normalizeLevel(raw: unknown): NocLogLine['level'] {
	const text = String(raw ?? '').toUpperCase();
	if (text.includes('ERROR') || text.includes('FATAL') || text.includes('CRIT')) {
		return 'ERROR';
	}
	if (text.includes('WARN')) {
		return 'WARN';
	}
	if (text.includes('DEBUG') || text.includes('TRACE')) {
		return 'DEBUG';
	}
	if (LEVELS.includes(text as NocLogLine['level'])) {
		return text as NocLogLine['level'];
	}
	return 'INFO';
}

function formatTs(timestamp: unknown): string {
	let ms: number | null = null;
	if (typeof timestamp === 'number') {
		// nanoseconds → milliseconds when the value is clearly ns-scale
		ms = timestamp > 1e15 ? timestamp / 1e6 : timestamp;
	} else if (typeof timestamp === 'string') {
		const parsed = Date.parse(timestamp);
		ms = Number.isNaN(parsed) ? null : parsed;
	}
	if (ms === null) {
		return '';
	}
	const d = new Date(ms);
	const pad = (n: number): string => String(n).padStart(2, '0');
	return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function pickService(data: Record<string, unknown>): string {
	const resources = (data.resources_string ?? {}) as Record<string, string>;
	const attrs = (data.attributes_string ?? {}) as Record<string, string>;
	return (
		resources['service.name'] ||
		attrs['service.name'] ||
		(data['service.name'] as string) ||
		''
	);
}

export interface UseNocLogsResult {
	logs: NocLogLine[];
	isLoading: boolean;
	isError: boolean;
}

export default function useNocLogs(limit = 7): UseNocLogsResult {
	const requestData = useMemo(
		() => cloneDeep(initialQueriesMap[DataSource.LOGS]),
		[],
	);

	const { data, isLoading, isError } = useGetExplorerQueryRange(
		requestData,
		PANEL_TYPES.LIST,
		ENTITY_VERSION_V5,
		{ enabled: true },
		undefined,
		false,
	);

	const logs = useMemo<NocLogLine[]>(() => {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const result = (data as any)?.payload?.data?.newResult?.data?.result;
		const list = Array.isArray(result) ? result[0]?.list : undefined;
		if (!Array.isArray(list)) {
			return [];
		}
		return list.slice(0, limit).map((item: Record<string, any>) => {
			const row = (item.data ?? {}) as Record<string, unknown>;
			return {
				ts: formatTs(item.timestamp ?? row.timestamp),
				level: normalizeLevel(row.severity_text ?? row.severityText),
				service: pickService(row),
				message: String(row.body ?? ''),
			};
		});
	}, [data, limit]);

	return { logs, isLoading, isError };
}
