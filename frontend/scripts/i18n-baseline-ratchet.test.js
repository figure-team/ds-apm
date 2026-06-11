// RED→GREEN for SCOPE DoD row 1: "하드코딩 영어 문자열 추가 시 → CI 가드 실패(baseline 초과)".
// The CI guard (check-hardcoded-strings.js --ci) must fail only on a finding that is
// NOT frozen in the baseline. That ratchet decision is pure set logic; it lives here so
// it can be unit-tested without the TypeScript AST scan (which needs node_modules).
//
// Run: node --test scripts/i18n-baseline-ratchet.test.js
import test from 'node:test';
import assert from 'node:assert/strict';
import { selectNewViolations, keyOf } from './i18n-baseline-ratchet.js';

const finding = (over = {}) => ({
	file: 'src/container/Foo/Foo.tsx',
	line: 12,
	kind: 'jsx-text',
	text: 'You are not sending traces yet.',
	...over,
});

test('flags a hardcoded string that is not in the baseline (guard must fail)', () => {
	const out = selectNewViolations([finding()], []);
	assert.equal(out.length, 1);
	assert.equal(out[0].text, 'You are not sending traces yet.');
});

test('ignores a finding already frozen in the baseline (debt does not block)', () => {
	const f = finding();
	const out = selectNewViolations([f], [keyOf(f)]);
	assert.equal(out.length, 0);
});

test('baseline identity ignores line numbers (reformatting must not trip the guard)', () => {
	const frozen = finding({ line: 12 });
	const moved = finding({ line: 480 });
	const out = selectNewViolations([moved], [keyOf(frozen)]);
	assert.equal(out.length, 0);
});

test('a new string in a baselined file is still flagged (per-string, not per-file)', () => {
	const frozen = finding({ text: 'Old frozen copy' });
	const fresh = finding({ text: 'Brand new hardcoded label', line: 99 });
	const out = selectNewViolations([frozen, fresh], [keyOf(frozen)]);
	assert.equal(out.length, 1);
	assert.equal(out[0].text, 'Brand new hardcoded label');
});

test('duplicate new findings are reported once (deduped within a run)', () => {
	const a = finding({ text: 'Repeated text' });
	const b = finding({ text: 'Repeated text', line: 77 });
	const out = selectNewViolations([a, b], []);
	assert.equal(out.length, 1);
});
