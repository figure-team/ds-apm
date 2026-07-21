import { Color } from '@signozhq/design-tokens';
import { colors } from 'lib/getRandomColor';
import { QueryData } from 'types/api/widgets/getQuery';
import { i18nText } from 'utils/i18nText';

// Function to determine if a color is "red-like" based on its RGB values
export function isRedLike(hex: string): boolean {
	const r = parseInt(hex.slice(1, 3), 16);
	const g = parseInt(hex.slice(3, 5), 16);
	const b = parseInt(hex.slice(5, 7), 16);
	return r > 180 && r > g * 1.4 && r > b * 1.4;
}

const SAFE_FALLBACK_COLORS = colors.filter((c) => !isRedLike(c));

const SEVERITY_VARIANT_COLORS: Record<string, string> = {
	TRACE: Color.BG_FOREST_600,
	Trace: Color.BG_FOREST_500,
	trace: Color.BG_FOREST_400,
	trc: Color.BG_FOREST_300,
	Trc: Color.BG_FOREST_200,

	DEBUG: Color.BG_AQUA_600,
	Debug: Color.BG_AQUA_500,
	debug: Color.BG_AQUA_400,
	dbg: Color.BG_AQUA_300,
	Dbg: Color.BG_AQUA_200,

	INFO: Color.BG_ROBIN_600,
	Info: Color.BG_ROBIN_500,
	info: Color.BG_ROBIN_400,
	Information: Color.BG_ROBIN_300,
	information: Color.BG_ROBIN_200,

	WARN: Color.BG_AMBER_600,
	Warn: Color.BG_AMBER_500,
	warn: Color.BG_AMBER_400,
	warning: Color.BG_AMBER_300,
	Warning: Color.BG_AMBER_200,
	wrn: Color.BG_AMBER_300,
	Wrn: Color.BG_AMBER_200,

	ERROR: Color.BG_CHERRY_600,
	Error: Color.BG_CHERRY_500,
	error: Color.BG_CHERRY_400,
	err: Color.BG_CHERRY_300,
	Err: Color.BG_CHERRY_200,
	ERR: Color.BG_CHERRY_600,
	fail: Color.BG_CHERRY_400,
	Fail: Color.BG_CHERRY_300,
	FAIL: Color.BG_CHERRY_600,

	FATAL: Color.BG_SAKURA_600,
	Fatal: Color.BG_SAKURA_500,
	fatal: Color.BG_SAKURA_400,
	critical: Color.BG_SAKURA_300,
	Critical: Color.BG_SAKURA_200,
	CRITICAL: Color.BG_SAKURA_600,
	crit: Color.BG_SAKURA_300,
	Crit: Color.BG_SAKURA_200,
	CRIT: Color.BG_SAKURA_600,
	panic: Color.BG_SAKURA_400,
	Panic: Color.BG_SAKURA_300,
	PANIC: Color.BG_SAKURA_600,
};

// 수집기·언어 런타임마다 표기가 달라(BG: .NET=Information, Go=info 등) 같은 의미의
// 심각도가 대소문자·별칭으로 갈라진다. 병합 기준은 대문자 정규형 + 아래 별칭 표.
const SEVERITY_CANONICAL_ALIASES: Record<string, string> = {
	TRC: 'TRACE',
	DBG: 'DEBUG',
	INFORMATION: 'INFO',
	WARNING: 'WARN',
	WRN: 'WARN',
	ERR: 'ERROR',
	FAIL: 'ERROR',
	CRIT: 'CRITICAL',
};

export const SEVERITY_NONE_LABEL_KEY = 'common:severity_none';

function canonicalSeverityLabel(rawLabel: string): string {
	const upper = rawLabel.trim().toUpperCase();
	if (!upper) {
		return i18nText(SEVERITY_NONE_LABEL_KEY);
	}
	return SEVERITY_CANONICAL_ALIASES[upper] || upper;
}

// severity_text로 그룹핑된 빈도 차트 시리즈를 정규형 기준으로 병합한다.
// severity_text 라벨이 없는 시리즈(라이브 로그의 단일 count 시리즈 등)는 그대로 통과.
export function normalizeFrequencyChartData(data: QueryData[]): QueryData[] {
	const mergedBySeverity = new Map<
		string,
		{ series: QueryData; sums: Map<number, number> }
	>();
	const result: QueryData[] = [];

	data.forEach((series) => {
		if (
			!series.metric ||
			!Object.prototype.hasOwnProperty.call(series.metric, 'severity_text')
		) {
			result.push(series);
			return;
		}

		const canonical = canonicalSeverityLabel(series.metric.severity_text || '');
		const existing = mergedBySeverity.get(canonical);
		const entry = existing ?? {
			series: {
				...series,
				metric: { ...series.metric, severity_text: canonical },
			},
			sums: new Map<number, number>(),
		};
		if (!existing) {
			mergedBySeverity.set(canonical, entry);
			result.push(entry.series);
		}

		(series.values || []).forEach(([timestamp, value]) => {
			const parsed = parseFloat(value);
			entry.sums.set(
				timestamp,
				(entry.sums.get(timestamp) || 0) + (Number.isNaN(parsed) ? 0 : parsed),
			);
		});
	});

	mergedBySeverity.forEach((entry) => {
		entry.series.values = Array.from(entry.sums.entries())
			.sort((a, b) => a[0] - b[0])
			.map(([timestamp, sum]) => [timestamp, String(sum)]);
	});

	return result;
}

// Simple function to get severity color for any component
export function getSeverityColor(severityText: string): string {
	const variantColor = SEVERITY_VARIANT_COLORS[severityText.trim()];
	if (variantColor) {
		return variantColor;
	}

	return Color.BG_ROBIN_500; // Default fallback
}

export function getColorsForSeverityLabels(
	label: string,
	index: number,
): string {
	const trimmed = label.trim();

	if (!trimmed || trimmed === i18nText(SEVERITY_NONE_LABEL_KEY)) {
		return Color.BG_VANILLA_400; // Default color for empty/no-severity labels
	}

	const variantColor = SEVERITY_VARIANT_COLORS[trimmed];
	if (variantColor) {
		return variantColor;
	}

	const lowerCaseLabel = label.toLowerCase();

	// Fallback to old format for backward compatibility
	if (lowerCaseLabel.includes(`{severity_text="trace"}`)) {
		return Color.BG_FOREST_400;
	}

	if (lowerCaseLabel.includes(`{severity_text="debug"}`)) {
		return Color.BG_AQUA_500;
	}

	if (
		lowerCaseLabel.includes(`{severity_text="info"}`) ||
		lowerCaseLabel.includes(`{severity_text=""}`)
	) {
		return Color.BG_ROBIN_500;
	}

	if (lowerCaseLabel.includes(`{severity_text="warn"}`)) {
		return Color.BG_AMBER_500;
	}

	if (lowerCaseLabel.includes(`{severity_text="error"}`)) {
		return Color.BG_CHERRY_500;
	}

	if (lowerCaseLabel.includes(`{severity_text="fatal"}`)) {
		return Color.BG_SAKURA_500;
	}

	return (
		SAFE_FALLBACK_COLORS[index % SAFE_FALLBACK_COLORS.length] ||
		Color.BG_VANILLA_400
	);
}
