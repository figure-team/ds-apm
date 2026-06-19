import getRun from 'api/codeRca/getRun';
import listRuns from 'api/codeRca/listRuns';
import { CodeRcaRunSummary } from 'api/codeRca/types';
import { useTranslation } from 'react-i18next';
import { useQuery } from 'react-query';

import { NocRca } from '../types';

export interface UseNocRcaResult {
	rca: NocRca | null;
	isLoading: boolean;
	isError: boolean;
}

function pickLatest(runs: CodeRcaRunSummary[]): CodeRcaRunSummary | null {
	if (runs.length === 0) {
		return null;
	}
	// prefer the newest completed run, otherwise the newest run overall
	const sorted = [...runs].sort((a, b) => b.createdAt - a.createdAt);
	return sorted.find((r) => r.status === 'done') ?? sorted[0];
}

export default function useNocRca(): UseNocRcaResult {
	const { t } = useTranslation('home');
	const {
		data: runs,
		isLoading: runsLoading,
		isError: runsError,
	} = useQuery({
		queryKey: ['noc-rca-runs'],
		queryFn: () => listRuns({ limit: 10 }),
		refetchOnWindowFocus: false,
	});

	const latest = pickLatest(runs?.data ?? []);

	const {
		data: detail,
		isLoading: detailLoading,
		isError: detailError,
	} = useQuery({
		queryKey: ['noc-rca-run', latest?.runId],
		queryFn: () => getRun(latest!.runId),
		enabled: Boolean(latest?.runId && latest?.status === 'done'),
		refetchOnWindowFocus: false,
	});

	if (!latest) {
		return {
			rca: null,
			isLoading: runsLoading,
			isError: runsError,
		};
	}

	const statusLabel = t(`noc_rca_status_${latest.status}`, latest.status).toString();

	const chips: string[] = [statusLabel];
	if (latest.baselineCommit) {
		chips.push(`commit ${latest.baselineCommit.slice(0, 7)}`);
	}
	if (detail?.data.confidence) {
		chips.push(`${detail.data.confidence}`);
	}

	const summary =
		detail?.data.rootCause?.trim() ||
		(latest.status === 'done'
			? t('noc_rca_summary_loading').toString()
			: statusLabel);

	const actions: string[] = [];
	if (detail?.data.proposedFix?.trim()) {
		actions.push(t('noc_rca_action_fix').toString());
	}

	return {
		rca: {
			title: t('noc_rca_title', { service: latest.service }).toString(),
			summary,
			chips,
			actions,
		},
		isLoading: runsLoading || detailLoading,
		isError: runsError || detailError,
	};
}
