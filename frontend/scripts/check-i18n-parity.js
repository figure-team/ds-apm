/* eslint-disable no-console */
// Verifies that every key in each en/<ns>.json has a counterpart in ko/<ns>.json.
// Usage:
//   node scripts/check-i18n-parity.js            # check all namespaces
//   node scripts/check-i18n-parity.js login      # check a single namespace
// Exit code 1 if any ko file is missing or any key set diverges.
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const EN_DIR = path.join(__dirname, '../public/locales/en');
const KO_DIR = path.join(__dirname, '../public/locales/ko');

const onlyNs = process.argv[2];
let failed = false;

// Flattens a (possibly nested) translation object into dot-path -> value pairs,
// so nested namespaces (e.g. trace.json's "options_menu") are validated deeply.
function flatten(obj, prefix = '') {
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

for (const file of fs.readdirSync(EN_DIR).filter((f) => f.endsWith('.json'))) {
	const ns = file.replace('.json', '');
	if (onlyNs && ns !== onlyNs) continue;

	const en = flatten(JSON.parse(fs.readFileSync(path.join(EN_DIR, file), 'utf8')));
	const koPath = path.join(KO_DIR, file);
	if (!fs.existsSync(koPath)) {
		console.error(`[${ns}] MISSING ko file`);
		failed = true;
		continue;
	}
	const ko = flatten(JSON.parse(fs.readFileSync(koPath, 'utf8')));
	const missing = Object.keys(en).filter((k) => !(k in ko));
	const extra = Object.keys(ko).filter((k) => !(k in en));
	const identical = Object.keys(en).filter(
		(k) => k in ko && en[k] === ko[k] && /[A-Za-z]/.test(String(en[k])),
	);

	if (missing.length || extra.length) {
		failed = true;
		console.error(
			`[${ns}] FAIL  missing(${missing.length}): ${missing.join(', ')}  |  extra(${extra.length}): ${extra.join(', ')}`,
		);
	} else if (identical.length) {
		console.warn(`[${ns}] WARN  ${identical.length} value(s) identical to EN: ${identical.join(', ')}`);
	} else {
		console.log(`[${ns}] OK (${Object.keys(ko).length} keys)`);
	}
}

process.exit(failed ? 1 : 0);
