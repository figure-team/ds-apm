import type { Page } from '@playwright/test';
import { test, expect } from '../../fixtures/auth';

// SCOPE DoD row 3 (T2 e2e):
//   Given the user switches language to ko
//   When the core screens (SideNav · login · dashboard) render
//   Then they show Korean copy and expose zero raw i18n keys (e.g. `a.b.c`).
//
// Language is selected through i18next's LanguageDetector, which reads the
// `i18nextLng` localStorage key (see src/ReactI18/index.tsx). Seeding it via an
// init script — before any document loads — is the deterministic equivalent of a
// user toggling the language in My Settings, without coupling to that page's DOM.
async function useKorean(page: Page): Promise<void> {
	await page.addInitScript(() => {
		window.localStorage.setItem('i18nextLng', 'ko');
	});
}

// Any Hangul syllable — proves the screen actually rendered Korean copy.
const HANGUL = /[ㄱ-힝]/;

// Returns visible leaf-text that looks like an untranslated i18n key rather than
// real copy. Tuned for low false positives: real UI copy has spaces, while raw
// keys are space-free snake_case (`button_login`), namespaced (`routes:home`),
// or dotted paths (`a.b.c`). Hostnames/versions/filenames are excluded.
async function findRawI18nKeys(page: Page): Promise<string[]> {
	return page.evaluate(() => {
		const looksLikeRawKey = (s: string): boolean => {
			if (/\s/.test(s) || s.length < 4 || s.length > 60) return false;
			if (/^[a-z][a-z0-9]*(?:_[a-z0-9]+)+$/.test(s)) return true; // snake_case key
			if (/^[a-zA-Z][\w$]*:[a-zA-Z0-9_.$]+$/.test(s)) return true; // ns:key
			if (/^[a-zA-Z][\w$]*(?:\.[a-zA-Z0-9_$]+)+$/.test(s)) {
				// dotted path: flag only multi-segment or snake-containing paths, and
				// never things that end like a domain/file (.com/.io/.json/...).
				if (/\.(json|js|jsx|ts|tsx|io|com|net|org|dev|cloud|ai|co)$/i.test(s)) return false;
				return s.includes('_') || s.split('.').length >= 3;
			}
			return false;
		};

		const out = new Set<string>();
		document.querySelectorAll('body *:not(script):not(style)').forEach((el) => {
			if (el.children.length !== 0) return; // leaf nodes only
			const text = (el.textContent || '').trim();
			if (text && looksLikeRawKey(text)) out.add(text);
		});
		return [...out];
	});
}

test.describe('i18n language switch → ko', () => {
	test('login screen renders Korean with no raw keys', async ({ page }) => {
		await useKorean(page);
		await page.goto('/login');

		// label_email → "이메일" (ko/login.json)
		await expect(page.getByText('이메일').first()).toBeVisible();
		expect(HANGUL.test((await page.locator('body').innerText()) || '')).toBe(true);

		const rawKeys = await findRawI18nKeys(page);
		expect(rawKeys, `raw i18n keys visible on login: ${rawKeys.join(', ')}`).toEqual([]);
	});

	test('SideNav renders Korean nav labels with no raw keys', async ({ authedPage: page }) => {
		await useKorean(page);
		await page.goto('/');

		// Nav labels come from the `routes` namespace (홈/대시보드/서비스/알림/로그/트레이스).
		await expect(
			page.getByText(/^(홈|대시보드|서비스|알림|로그|트레이스)$/).first(),
		).toBeVisible();

		const rawKeys = await findRawI18nKeys(page);
		expect(rawKeys, `raw i18n keys visible on home/SideNav: ${rawKeys.join(', ')}`).toEqual([]);
	});

	test('dashboards screen renders Korean with no raw keys', async ({ authedPage: page }) => {
		await useKorean(page);
		await page.goto('/dashboard');

		// titles ns → document title localizes ("SigNoz | 전체 대시보드").
		await expect(page).toHaveTitle(/대시보드/);

		const rawKeys = await findRawI18nKeys(page);
		expect(rawKeys, `raw i18n keys visible on dashboards: ${rawKeys.join(', ')}`).toEqual([]);
	});
});
