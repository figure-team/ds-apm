/* eslint-disable no-console */
// Verifies that every key in each en/<ns>.json has a counterpart in ko/<ns>.json
// and that no value is blank. Usage:
//   node scripts/check-i18n-parity.js            # check all namespaces
//   node scripts/check-i18n-parity.js login      # check a single namespace
// Exit code 1 if any ko file is missing, any key set diverges, or any value is empty.
// The comparison itself lives in i18n-parity.js (pure, unit-tested).
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { compareNamespace } from './i18n-parity.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const EN_DIR = path.join(__dirname, '../public/locales/en');
const KO_DIR = path.join(__dirname, '../public/locales/ko');

const onlyNs = process.argv[2];
let failed = false;

for (const file of fs.readdirSync(EN_DIR).filter((f) => f.endsWith('.json'))) {
	const ns = file.replace('.json', '');
	if (onlyNs && ns !== onlyNs) continue;

	const en = JSON.parse(fs.readFileSync(path.join(EN_DIR, file), 'utf8'));
	const koPath = path.join(KO_DIR, file);
	if (!fs.existsSync(koPath)) {
		console.error(`[${ns}] MISSING ko file`);
		failed = true;
		continue;
	}
	const ko = JSON.parse(fs.readFileSync(koPath, 'utf8'));
	const { missing, extra, empty, identical, ok, keyCount } = compareNamespace(en, ko);

	if (!ok) {
		failed = true;
		console.error(
			`[${ns}] FAIL  missing(${missing.length}): ${missing.join(', ')}  |  ` +
				`extra(${extra.length}): ${extra.join(', ')}  |  ` +
				`empty(${empty.length}): ${empty.join(', ')}`,
		);
	} else if (identical.length) {
		console.warn(`[${ns}] WARN  ${identical.length} value(s) identical to EN: ${identical.join(', ')}`);
	} else {
		console.log(`[${ns}] OK (${keyCount} keys)`);
	}
}

process.exit(failed ? 1 : 0);
