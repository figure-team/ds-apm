import { useListRules } from 'api/generated/services/rules';
import type { RuletypesRuleDTO } from 'api/generated/services/sigNoz.schemas';
import type { TFunction } from 'i18next';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import { NocAlert, NocSeverity } from '../types';

const SEVERITY_MAP: Record<string, NocSeverity> = {
	critical: 'critical',
	fatal: 'critical',
	error: 'error',
	high: 'error',
	warning: 'warning',
	warn: 'warning',
	medium: 'warning',
	info: 'info',
	low: 'info',
};

function toSeverity(rule: RuletypesRuleDTO): NocSeverity {
	const raw = (rule.labels?.severity ?? '').toLowerCase();
	return SEVERITY_MAP[raw] ?? 'warning';
}

function relativeAge(t: TFunction, updatedAt?: Date): string {
	if (!updatedAt) {
		return '';
	}
	const then = new Date(updatedAt).getTime();
	if (Number.isNaN(then)) {
		return '';
	}
	const diffMs = Date.now() - then;
	const min = Math.floor(diffMs / 60000);
	if (min < 1) {
		return t('noc_age_now').toString();
	}
	if (min < 60) {
		return t('noc_age_min', { count: min }).toString();
	}
	const hours = Math.floor(min / 60);
	if (hours < 24) {
		return t('noc_age_hour', { count: hours }).toString();
	}
	return t('noc_age_day', { count: Math.floor(hours / 24) }).toString();
}

const SEVERITY_RANK: Record<NocSeverity, number> = {
	critical: 0,
	error: 1,
	warning: 2,
	info: 3,
};

export interface UseNocAlertsResult {
	alerts: NocAlert[];
	firingCount: number;
	totalCount: number;
	isLoading: boolean;
	isError: boolean;
	/** 최근 해소된 알림 이력 — AlertsPanel 빈 상태용. 계산은 Lane A(impl-plan Task 9 Step 4). */
	lastResolved?: { age: string; service: string };
}

export default function useNocAlerts(limit = 6): UseNocAlertsResult {
	const { t } = useTranslation('home');
	const { data, isLoading, isError } = useListRules({ query: { cacheTime: 0 } });

	return useMemo(() => {
		const rules = data?.data ?? [];
		const firingCount = rules.filter((rule) => rule.state === 'firing').length;

		const sorted = [...rules].sort((a, b) => {
			// firing alerts first
			if (a.state === 'firing' && b.state !== 'firing') {
				return -1;
			}
			if (a.state !== 'firing' && b.state === 'firing') {
				return 1;
			}
			// then by severity
			const rankDelta = SEVERITY_RANK[toSeverity(a)] - SEVERITY_RANK[toSeverity(b)];
			if (rankDelta !== 0) {
				return rankDelta;
			}
			// then most recently updated
			return (
				new Date(b.updatedAt ?? 0).getTime() - new Date(a.updatedAt ?? 0).getTime()
			);
		});

		const alerts: NocAlert[] = sorted.slice(0, limit).map((rule) => ({
			id: rule.id,
			severity: toSeverity(rule),
			title: rule.alert,
			meta: rule.description?.trim() || (rule.labels?.severity ?? ''),
			age: relativeAge(t, rule.updatedAt),
		}));

		// 최근 해소 이력: firing이 아닌 규칙 중 가장 최근 updatedAt (AlertsPanel 빈 상태용).
		const resolvedRule = [...rules]
			.filter((r) => r.state !== 'firing')
			.sort(
				(a, b) =>
					new Date(b.updatedAt ?? 0).getTime() - new Date(a.updatedAt ?? 0).getTime(),
			)[0];
		const lastResolved = resolvedRule
			? {
					age: relativeAge(t, resolvedRule.updatedAt),
					service: resolvedRule.labels?.service_name ?? resolvedRule.alert ?? '',
			  }
			: undefined;

		return {
			alerts,
			firingCount,
			totalCount: rules.length,
			isLoading,
			isError,
			lastResolved,
		};
	}, [data, isLoading, isError, limit, t]);
}
