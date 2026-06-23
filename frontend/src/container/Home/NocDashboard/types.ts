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
