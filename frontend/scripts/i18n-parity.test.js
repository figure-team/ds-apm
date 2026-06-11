// RED→GREEN for SCOPE DoD row 2: "ko/en 키 누락·빈 값 → 패리티 실패".
// The parity comparison is pure (flatten + set diff + empty detection); it lives in
// i18n-parity.js so it can be unit-tested without touching the filesystem CLI.
//
// Run: node --test scripts/i18n-parity.test.js
import test from 'node:test';
import assert from 'node:assert/strict';
import { compareNamespace, flatten } from './i18n-parity.js';

test('matching keys with real translations pass parity', () => {
	const r = compareNamespace({ greeting: 'Hello' }, { greeting: '안녕' });
	assert.equal(r.ok, true);
	assert.deepEqual(r.missing, []);
	assert.deepEqual(r.extra, []);
	assert.deepEqual(r.empty, []);
});

test('a ko key with an empty string value fails parity', () => {
	const r = compareNamespace({ greeting: 'Hello' }, { greeting: '' });
	assert.deepEqual(r.empty, ['greeting']);
	assert.equal(r.ok, false);
});

test('a ko key with a whitespace-only value fails parity', () => {
	const r = compareNamespace({ greeting: 'Hello' }, { greeting: '   ' });
	assert.deepEqual(r.empty, ['greeting']);
	assert.equal(r.ok, false);
});

test('an en key missing from ko fails parity', () => {
	const r = compareNamespace({ a: 'A', b: 'B' }, { a: '에이' });
	assert.deepEqual(r.missing, ['b']);
	assert.equal(r.ok, false);
});

test('an extra ko key absent from en fails parity', () => {
	const r = compareNamespace({ a: 'A' }, { a: '에이', z: '지' });
	assert.deepEqual(r.extra, ['z']);
	assert.equal(r.ok, false);
});

test('nested namespaces are validated deeply via dot-paths', () => {
	const r = compareNamespace({ menu: { open: 'Open' } }, { menu: { open: '' } });
	assert.deepEqual(r.empty, ['menu.open']);
	assert.equal(r.ok, false);
});

test('identical latin values are a non-fatal warning, not a failure', () => {
	const r = compareNamespace({ brand: 'SigNoz' }, { brand: 'SigNoz' });
	assert.deepEqual(r.identical, ['brand']);
	assert.equal(r.ok, true);
});

test('flatten turns nested objects into dot-path keys', () => {
	assert.deepEqual(flatten({ a: { b: { c: 'x' } } }), { 'a.b.c': 'x' });
});
