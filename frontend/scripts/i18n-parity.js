// Pure ko/en parity comparison (SCOPE DoD row 2), extracted from check-i18n-parity.js
// so the key-diff + empty-value rules are unit-testable without filesystem I/O.

// Flattens a (possibly nested) translation object into dot-path -> value pairs,
// so nested namespaces (e.g. trace.json's "options_menu") are validated deeply.
export function flatten(obj, prefix = '') {
	const out = {};
	for (const [key, value] of Object.entries(obj)) {
		const full = prefix ? `${prefix}.${key}` : key;
		if (value && typeof value === 'object' && !Array.isArray(value)) {
			Object.assign(out, flatten(value, full));
		} else {
			out[full] = value;
		}
	}
	return out;
}

// A translation value is "empty" when it is null/undefined or only whitespace —
// a present-but-blank value is as broken as a missing key for the user.
const isEmpty = (v) => v == null || (typeof v === 'string' && v.trim() === '');

// Compares one namespace's en/ko objects (raw or nested). Returns the diff plus an
// `ok` flag that is true only when keys match on both sides and no value is blank.
// `identical` (same latin text on both sides) is a warning, not a failure.
export function compareNamespace(enObj, koObj) {
	const en = flatten(enObj);
	const ko = flatten(koObj);

	const missing = Object.keys(en).filter((k) => !(k in ko));
	const extra = Object.keys(ko).filter((k) => !(k in en));
	const empty = [
		...new Set([
			...Object.keys(en).filter((k) => isEmpty(en[k])),
			...Object.keys(ko).filter((k) => isEmpty(ko[k])),
		]),
	].sort();
	const identical = Object.keys(en).filter(
		(k) => k in ko && en[k] === ko[k] && /[A-Za-z]/.test(String(en[k])),
	);

	const ok = missing.length === 0 && extra.length === 0 && empty.length === 0;
	return { missing, extra, empty, identical, ok, keyCount: Object.keys(ko).length };
}
