import { getHostLists, HostData } from 'api/infraMonitoring/getHostLists';
import { useMemo } from 'react';
import { useQuery } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { AppState } from 'store/reducers';
import { GlobalReducer } from 'types/reducer/globalTime';

import { NocHealth, NocInfraHost } from '../types';

const HOST_LIMIT = 12;
const CPU_WARN = 65;
const CPU_CRIT = 90;

function healthFromCpu(cpuPct: number): NocHealth {
	if (cpuPct >= CPU_CRIT) return 'critical';
	if (cpuPct >= CPU_WARN) return 'warning';
	return 'healthy';
}

// cpu/memory는 분수(0–1) — 100× 정규화 (§5.2/§9)
export function mapHost(
	r: Pick<HostData, 'hostName' | 'cpu' | 'memory' | 'active'>,
): NocInfraHost {
	const cpu = Math.round((Number.isFinite(r.cpu) ? r.cpu : 0) * 100);
	const mem = Math.round((Number.isFinite(r.memory) ? r.memory : 0) * 100);
	return { name: r.hostName, cpu, mem, health: healthFromCpu(cpu) };
}

export interface UseNocInfraResult {
	hosts: NocInfraHost[];
	isLoading: boolean;
	isError: boolean;
}

export default function useNocInfra(): UseNocInfraResult {
	const { maxTime, minTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);

	const { data, isLoading, isError } = useQuery(
		['noc-infra-hosts', minTime, maxTime],
		async () => {
			const res = await getHostLists({
				filters: { items: [], op: 'AND' },
				groupBy: [],
				orderBy: { columnName: 'cpu', order: 'desc' },
				limit: HOST_LIMIT,
				start: Math.floor(minTime / 1e6),
				end: Math.floor(maxTime / 1e6),
			});
			if ('payload' in res && res.payload) {
				return res.payload.data.records;
			}
			throw new Error('host list failed');
		},
		{ keepPreviousData: true },
	);

	const hosts = useMemo<NocInfraHost[]>(
		() => (data ?? []).slice(0, HOST_LIMIT).map(mapHost),
		[data],
	);

	return { hosts, isLoading, isError };
}
