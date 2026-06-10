import { SearchOutlined } from '@ant-design/icons';
import { Button, Card, Input, Space } from 'antd';
import type { FilterDropdownProps } from 'antd/es/table/interface';
import { TFunction } from 'i18next';

export const getFilterDropdown = (t: TFunction) => ({
	setSelectedKeys,
	selectedKeys,
	confirm,
}: FilterDropdownProps): JSX.Element => {
	const handleSearch = (): void => {
		confirm();
	};

	const selectedKeysHandler = (e: React.ChangeEvent<HTMLInputElement>): void => {
		setSelectedKeys(e.target.value ? [e.target.value] : []);
	};

	return (
		<Card size="small">
			<Space align="start" direction="vertical">
				<Input
					placeholder={t('search_by_service').toString()}
					value={selectedKeys[0]}
					onChange={selectedKeysHandler}
					allowClear
					onPressEnter={handleSearch}
				/>
				<Button
					type="primary"
					onClick={handleSearch}
					icon={<SearchOutlined />}
					size="small"
				>
					{t('search').toString()}
				</Button>
			</Space>
		</Card>
	);
};
