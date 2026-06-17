import type { Labels } from 'types/api/alerts/def';

export type RequiredOperationalLabel = {
	key: string;
	label: string;
	description: string;
};

export const RECOMMENDED_OPERATIONAL_LABELS: RequiredOperationalLabel[] = [
	{
		key: 'project_id',
		label: 'Project ID',
		description: 'SOP 테넌트 범위 검증·AI 전략 org 선택에 사용. environment와 함께 없으면 SOP 문서 접근이 차단됩니다.',
	},
	{
		key: 'environment',
		label: 'Environment',
		description: 'project_id와 함께 SOP 테넌트 정책을 완성합니다. 둘 다 있어야 SOP 연동이 활성화됩니다.',
	},
	{
		key: 'service.name',
		label: 'Service Name',
		description: '알림 발송 payload(Slack·PagerDuty·webhook)에 Service 필드로 포함됩니다.',
	},
	{
		key: 'owner_team',
		label: 'Owner Team',
		description: '알림 발송 payload에 Owner team 필드로 포함됩니다.',
	},
	// severity는 더 이상 수동 라벨이 아니다: 임계값(threshold) 이름이 곧 severity이며
	// 발화 시 severity 라벨로 자동 파생되고 라우팅 정책 매칭(threshold.name)에 쓰인다.
	// 정적 라벨로 추천하면 자동 파생값을 덮어써 라우팅/표시 등급이 어긋나므로 제외한다.
];

export function getMissingOperationalLabels(
	labels: Labels,
): RequiredOperationalLabel[] {
	return RECOMMENDED_OPERATIONAL_LABELS.filter(({ key }) => !labels[key]?.trim());
}

export function hasOperationalLabel(labels: Labels, key: string): boolean {
	return Boolean(labels[key]?.trim());
}
