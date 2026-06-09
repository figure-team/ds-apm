import { useTranslation } from 'react-i18next';
import { Button, Row } from 'antd';

interface SearchFieldsActionBarProps {
	applyUpdate: VoidFunction;
	clearFilters: VoidFunction;
}

export function SearchFieldsActionBar({
	applyUpdate,
	clearFilters,
}: SearchFieldsActionBarProps): JSX.Element | null {
	const { t } = useTranslation(['logs']);
	return (
		<Row style={{ justifyContent: 'flex-end', paddingRight: '2.4rem' }}>
			<Button
				type="default"
				onClick={clearFilters}
				style={{ marginRight: '1rem' }}
			>
				{t('logs:clear_filter')}
			</Button>
			<Button type="primary" onClick={applyUpdate}>
				{t('logs:apply')}
			</Button>
		</Row>
	);
}
export default SearchFieldsActionBar;
