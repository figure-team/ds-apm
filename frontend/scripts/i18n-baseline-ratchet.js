// Baseline ratchet for the hardcoded-string guard (SCOPE DoD row 1).
// Pure set logic, extracted from check-hardcoded-strings.js so it is unit-testable
// without the TypeScript AST scan. The CI guard fails only on findings selected here.

// Baseline identity ignores line numbers (which shift on unrelated edits) so the
// ratchet only reacts to genuinely new strings, not reformatting.
export const keyOf = (f) => `${f.file}\t${f.kind}\t${f.text}`;

// Returns the findings that are NOT frozen in the baseline, deduped so each unique
// (file, kind, text) is reported at most once per run.
export function selectNewViolations(findings, baseline) {
	const base = baseline instanceof Set ? baseline : new Set(baseline);
	const seen = new Set();
	return findings.filter((f) => {
		const k = keyOf(f);
		if (base.has(k) || seen.has(k)) return false;
		seen.add(k);
		return true;
	});
}
