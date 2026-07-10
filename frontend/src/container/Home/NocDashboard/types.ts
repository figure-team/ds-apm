import { Widgets } from 'types/api/dashboard/getAll';

export type NocSeverity = 'critical' | 'error' | 'warning' | 'info';

export type NocHealth = 'healthy' | 'warning' | 'critical';

export type NocAccent = 'brand' | 'error' | 'ok' | 'neutral';

export interface NocKpi {
	key: string;
	label: string;
	value: string;
	unit?: string;
	delta?: string;
	deltaDir?: 'up' | 'down' | 'flat';
	accent?: NocAccent;
	/** normalized 0..1 points for the mini sparkline */
	spark?: number[];
	sparkColor?: string;
}

export interface NocAlert {
	id: string;
	severity: NocSeverity;
	title: string;
	meta: string;
	age: string;
}

export interface NocServiceRow {
	name: string;
	health: NocHealth;
	p99Ms: number;
	errPct: number;
	rps: number;
}

export interface NocLogLine {
	ts: string;
	level: 'ERROR' | 'WARN' | 'INFO' | 'DEBUG';
	service: string;
	message: string;
}

export interface NocRca {
	title: string;
	summary: string;
	chips: string[];
	actions: string[];
}

// ===== C-2 재구조 (트리아지 + 멀티서비스 트렌드) 계약 타입 =====

export type TrendMetric = 'err' | 'p99' | 'rps';

export interface TrendPoint {
	t: number; // epoch ms
	v: number;
}

export interface TrendSeries {
	name: string;
	color: string;
	points: TrendPoint[];
	/** true면 대상이었으나 데이터 없음 — 선 생략, 범례 회색 표기 (§6.1) */
	missing?: boolean;
}

/** 컨테이너가 다크모드 팔레트를 반영해 색을 확정한 표시용 계열 */
export interface ResolvedTrendSeries extends TrendSeries {
	resolvedColor: string;
}

export interface NocInfraHost {
	name: string;
	cpu: number; // percent 0..100 (hosts/list 분수값 100× 정규화 후)
	mem: number; // percent 0..100
	health: NocHealth;
}

export interface NocCounts {
	critical: number;
	warning: number;
	healthy: number;
	alerts: number;
}

export interface WatchSelection {
	services: NocServiceRow[];
	mode: 'anomaly' | 'watch';
}

export interface TrendTarget {
	name: string;
	color: string;
}

// ===== v4 사용자 고정 패널 =====

export interface NocPinnedRef {
	dashboardId: string;
	widgetId: string;
}

export interface NocPinnedSlot {
	ref: NocPinnedRef;
	/** 원본 대시보드 제목 — 카드 캡션용. 대시보드 삭제 시 '' */
	dashboardTitle: string;
	/** 쿼리 엔티티 버전 — GridCardGraph version prop */
	version: string;
	/** null = 원본 대시보드/위젯이 삭제되었거나 핀 불가로 변경됨 */
	widget: Widgets | null;
}
