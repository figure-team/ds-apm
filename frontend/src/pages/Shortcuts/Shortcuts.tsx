import { Table, Typography } from 'antd';
import { useTranslation } from 'react-i18next';

import {
	ALL_SHORTCUTS,
	generateTableData,
	getShortcutColumns,
	SHORTCUT_SECTION_META,
} from './utils';

import './Shortcuts.styles.scss';

function Shortcuts(): JSX.Element {
	const { t } = useTranslation(['shortcuts']);

	function getShortcutTable(shortcutSection: string): JSX.Element {
		const tableData = generateTableData(shortcutSection, t);

		return (
			<section className="shortcut-section">
				<Typography.Text className="shortcut-section-heading">
					{t(SHORTCUT_SECTION_META[shortcutSection].labelKey)}
				</Typography.Text>
				<Table
					columns={getShortcutColumns(t)}
					dataSource={tableData}
					pagination={false}
					className="shortcut-section-table"
					bordered
				/>
			</section>
		);
	}

	return (
		<div className="keyboard-shortcuts">
			{Object.keys(ALL_SHORTCUTS).map((shortcutSection) =>
				getShortcutTable(shortcutSection),
			)}
		</div>
	);
}

export default Shortcuts;
