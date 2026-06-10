import { TableProps } from 'antd';
import { TFunction } from 'i18next';
import {
	DashboardShortcuts,
	DashboardShortcutsName,
} from 'constants/shortcuts/DashboardShortcuts';
import {
	GlobalShortcuts,
	GlobalShortcutsName,
} from 'constants/shortcuts/globalShortcuts';
import {
	LogsExplorerShortcuts,
	LogsExplorerShortcutsName,
} from 'constants/shortcuts/logsExplorerShortcuts';
import { QBShortcuts, QBShortcutsName } from 'constants/shortcuts/QBShortcuts';

export const ALL_SHORTCUTS: Record<string, Record<string, string>> = {
	'Global Shortcuts': GlobalShortcuts,
	'Logs Explorer Shortcuts': LogsExplorerShortcuts,
	'Query Builder Shortcuts': QBShortcuts,
	'Dashboard Shortcuts': DashboardShortcuts,
};

export const ALL_SHORTCUTS_LABEL: Record<string, Record<string, string>> = {
	'Global Shortcuts': GlobalShortcutsName,
	'Logs Explorer Shortcuts': LogsExplorerShortcutsName,
	'Query Builder Shortcuts': QBShortcutsName,
	'Dashboard Shortcuts': DashboardShortcutsName,
};

// Maps each section (used as the ALL_SHORTCUTS key) to its i18n heading key and
// the prefix for its per-shortcut description keys in shortcuts.json.
export const SHORTCUT_SECTION_META: Record<
	string,
	{ labelKey: string; prefix: string }
> = {
	'Global Shortcuts': { labelKey: 'section_global', prefix: 'global' },
	'Logs Explorer Shortcuts': {
		labelKey: 'section_logs_explorer',
		prefix: 'logs',
	},
	'Query Builder Shortcuts': { labelKey: 'section_query_builder', prefix: 'qb' },
	'Dashboard Shortcuts': { labelKey: 'section_dashboard', prefix: 'dashboard' },
};

interface ShortcutRow {
	shortcutKey: string;
	shortcutDescription: string;
}

export const getShortcutColumns = (
	t: TFunction,
): TableProps<ShortcutRow>['columns'] => [
	{
		title: t('shortcuts:col_shortcut').toString(),
		dataIndex: 'shortcutKey',
		key: 'shortcutKey',
		width: '30%',
		className: 'shortcut-key',
	},
	{
		title: t('shortcuts:col_description').toString(),
		dataIndex: 'shortcutDescription',
		key: 'shortcutDescription',
		className: 'shortcut-description',
	},
];

export function generateTableData(
	shortcutSection: string,
	t: TFunction,
): TableProps<ShortcutRow>['dataSource'] {
	const shortcuts = ALL_SHORTCUTS[shortcutSection];
	const shortcutsLabel = ALL_SHORTCUTS_LABEL[shortcutSection];
	const { prefix } = SHORTCUT_SECTION_META[shortcutSection];
	return Object.keys(shortcuts).map((shortcutName) => ({
		key: `${shortcuts[shortcutName]} ${shortcutName}`,
		shortcutKey: shortcutsLabel[shortcutName],
		shortcutDescription: t(`shortcuts:${prefix}_${shortcutName}`).toString(),
	}));
}
