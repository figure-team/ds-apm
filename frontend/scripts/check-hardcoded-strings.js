/* eslint-disable no-console */
// Detects hardcoded user-facing English strings in JSX that should go through i18n (t()).
// Uses the TypeScript AST (not regex) so className/URLs/identifiers don't produce false positives.
//
// Usage:
//   node scripts/check-hardcoded-strings.js                   # report every finding, exit 0
//   node scripts/check-hardcoded-strings.js --summary         # per-file counts only
//   node scripts/check-hardcoded-strings.js --update-baseline # freeze current findings as the baseline
//   node scripts/check-hardcoded-strings.js --ci              # exit 1 only on findings NOT in the baseline
//
// Baseline ratchet: pre-existing hardcoded strings are frozen in i18n-hardcoded-baseline.json.
// CI fails only when a NEW one is introduced, so the debt burns down without blocking on all of it.
//
// What it flags:
//   1. JSX text nodes        <div>You are not sending traces yet.</div>
//   2. String literals on user-facing attributes (title, placeholder, label, alt, ...)
// Already-translated values like {t('...')} are JSX expressions, not literals, so they are skipped.
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import ts from 'typescript';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC_DIR = path.join(__dirname, '../src');
const BASELINE_PATH = path.join(__dirname, 'i18n-hardcoded-baseline.json');

const ciMode = process.argv.includes('--ci');
const summaryMode = process.argv.includes('--summary');
const updateBaseline = process.argv.includes('--update-baseline');

// Baseline identity ignores line numbers (which shift on unrelated edits) so the
// ratchet only reacts to genuinely new strings, not reformatting.
const keyOf = (f) => `${f.file}\t${f.kind}\t${f.text}`;

// Attributes whose literal string values are shown to the user.
const USER_FACING_ATTRS = new Set([
	'title',
	'placeholder',
	'label',
	'alt',
	'description',
	'tooltip',
	'header',
	'subtitle',
	'subTitle',
	'caption',
	'message',
	'okText',
	'cancelText',
	'ariaLabel',
	'aria-label',
]);

// Skip non-product code.
function isExcluded(file) {
	return (
		/\.(test|spec|stories)\.tsx$/.test(file) ||
		/[/\\](__tests__|__mocks__|tests|mocks)[/\\]/.test(file) ||
		/[/\\]api[/\\]generated[/\\]/.test(file)
	);
}

// True only when the text is a real, user-readable phrase — strips HTML entities first,
// then requires at least two consecutive letters so stray symbols/numbers are ignored.
// Excludes technical tokens that are never translated copy: emails, URLs, and
// kebab/snake-case identifiers (e.g. "empty-alert-icon", "correlation-graphic").
function isMeaningfulText(raw) {
	const text = raw.replace(/&[a-zA-Z]+;|&#\d+;/g, ' ').trim();
	if (!/[A-Za-z]{2,}/.test(text)) return false;
	if (/^\S+@\S+\.\S+$/.test(text)) return false; // email
	if (/^https?:\/\//.test(text)) return false; // url
	if (/^[a-z0-9]+([-_][a-z0-9]+)+$/.test(text)) return false; // kebab/snake identifier
	return true;
}

function* walk(dir) {
	for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
		const full = path.join(dir, entry.name);
		if (entry.isDirectory()) {
			yield* walk(full);
		} else if (entry.name.endsWith('.tsx')) {
			yield full;
		}
	}
}

const findings = []; // { file, line, kind, text }

for (const file of walk(SRC_DIR)) {
	if (isExcluded(file)) continue;

	const source = fs.readFileSync(file, 'utf8');
	const sf = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
	const rel = path.relative(path.join(__dirname, '..'), file).replace(/\\/g, '/');

	const record = (node, kind, text) => {
		const { line } = sf.getLineAndCharacterOfPosition(node.getStart(sf));
		findings.push({ file: rel, line: line + 1, kind, text: text.trim().replace(/\s+/g, ' ') });
	};

	const visit = (node) => {
		if (ts.isJsxText(node)) {
			if (isMeaningfulText(node.text)) record(node, 'jsx-text', node.text);
		} else if (ts.isJsxAttribute(node) && node.initializer) {
			const name = node.name.getText(sf);
			if (
				USER_FACING_ATTRS.has(name) &&
				ts.isStringLiteral(node.initializer) &&
				isMeaningfulText(node.initializer.text)
			) {
				record(node, `attr:${name}`, node.initializer.text);
			}
		}
		ts.forEachChild(node, visit);
	};
	visit(sf);
}

findings.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);

// ---- --update-baseline: freeze current findings ----
if (updateBaseline) {
	const keys = [...new Set(findings.map(keyOf))].sort();
	fs.writeFileSync(BASELINE_PATH, `${JSON.stringify(keys, null, 2)}\n`);
	console.log(`Baseline written: ${keys.length} entr(ies) -> ${path.relative(process.cwd(), BASELINE_PATH)}`);
	process.exit(0);
}

// ---- --ci: fail only on findings missing from the baseline ----
if (ciMode) {
	const baseline = fs.existsSync(BASELINE_PATH)
		? new Set(JSON.parse(fs.readFileSync(BASELINE_PATH, 'utf8')))
		: new Set();
	const seen = new Set();
	const newViolations = findings.filter((f) => {
		const k = keyOf(f);
		if (baseline.has(k) || seen.has(k)) return false;
		seen.add(k);
		return true;
	});

	if (newViolations.length) {
		console.error('New hardcoded user-facing string(s) detected (not in baseline):\n');
		for (const f of newViolations) {
			console.error(`  ${f.file}:${f.line}\t[${f.kind}] ${f.text}`);
		}
		console.error(
			`\n=== ${newViolations.length} new violation(s) ===\n` +
				'Wrap them with i18n t() (see src/container/NoLogs/NoLogs.tsx),\n' +
				'or, if intentional, run `yarn i18n:check-literals:update` to update the baseline.',
		);
		process.exit(1);
	}
	console.log('No new hardcoded strings beyond baseline. OK.');
	process.exit(0);
}

// ---- default / --summary: report every finding ----
const byFile = new Map();
for (const f of findings) {
	if (!byFile.has(f.file)) byFile.set(f.file, []);
	byFile.get(f.file).push(f);
}

if (summaryMode) {
	for (const [file, items] of byFile) {
		console.log(`${items.length}\t${file}`);
	}
} else {
	for (const [file, items] of byFile) {
		console.log(`\n${file}`);
		for (const f of items) {
			console.log(`  ${f.line}\t[${f.kind}] ${f.text}`);
		}
	}
}

console.log(
	`\n=== ${findings.length} hardcoded string(s) in ${byFile.size} file(s) ===`,
);
process.exit(0);
