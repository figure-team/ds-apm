import {
	settingsNavItemKeyMap,
	settingsSectionTitleKeyMap,
	settingsNavSections,
} from 'container/SideNav/menuItems';
import ROUTES from 'constants/routes';

describe('settingsNavItemKeyMap', () => {
	it('maps known item keys to i18n keys', () => {
		expect(settingsNavItemKeyMap['account']).toBe('routes:account');
		expect(settingsNavItemKeyMap['workspace']).toBe('routes:data_retention');
		expect(settingsNavItemKeyMap['manage-license']).toBe('settings:manage_license');
		expect(settingsNavItemKeyMap['keyboard-shortcuts']).toBe(
			'routes:keyboard_shortcuts',
		);
	});
});

describe('settingsSectionTitleKeyMap', () => {
	it('maps every section key to an i18n key', () => {
		settingsNavSections.forEach((section) => {
			expect(settingsSectionTitleKeyMap[section.key]).toBeDefined();
		});
	});

	it('maps section keys to i18n keys', () => {
		expect(settingsSectionTitleKeyMap['identity-access']).toBe(
			'routes:identity_access',
		);
		expect(settingsSectionTitleKeyMap['authentication']).toBe(
			'routes:authentication',
		);
	});
});

describe('settingsNavSections', () => {
	it('includes manage-license item in billing-license section', () => {
		const billing = settingsNavSections.find((s) => s.key === 'billing-license');
		const manageLicense = billing?.items.find(
			(i) => i.itemKey === 'manage-license',
		);
		expect(manageLicense).toBeDefined();
		expect(manageLicense?.key).toBe(ROUTES.LIST_LICENSES);
		expect(manageLicense?.isEnabled).toBe(false);
	});

	it('preserves all expected sections', () => {
		const keys = settingsNavSections.map((s) => s.key);
		expect(keys).toContain('data');
		expect(keys).toContain('ai-automation');
		expect(keys).toContain('alerts');
		expect(keys).toContain('identity-access');
		expect(keys).toContain('authentication');
		expect(keys).toContain('billing-license');
		expect(keys).toContain('personal');
	});

	it('keeps every settings item exactly once across sections', () => {
		const itemKeys = settingsNavSections.flatMap((s) =>
			s.items.map((i) => i.itemKey as string),
		);
		expect(new Set(itemKeys).size).toBe(itemKeys.length);
		expect(itemKeys).toHaveLength(18);
	});
});
