import { ChangeEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Button, Input, Popover, Tooltip, Typography } from 'antd';
import { ArrowDownWideNarrow, Check, Plus, Search } from 'lucide-react';
import { useAppContext } from 'providers/App/App';

interface SearchBarProps {
	searchQuery: string;
	sortOrder: {
		columnKey: string;
		order: 'ascend' | 'descend';
	};
	onSearch: (e: ChangeEvent<HTMLInputElement>) => void;
	onSort: (key: string) => void;
	onCreateFunnel: () => void;
}

function SearchBar({
	searchQuery,
	sortOrder,
	onSearch,
	onSort,
	onCreateFunnel,
}: SearchBarProps): JSX.Element {
	const { t } = useTranslation('trace');
	const { hasEditPermission } = useAppContext();

	return (
		<div className="search">
			<Popover
				trigger="click"
				content={
					<div className="sort-popover-content">
						<Typography.Text className="sort-popover-content__heading">
							{t('funnels.sort_by')}
						</Typography.Text>
						<Button
							type="text"
							className="sort-popover-content__button"
							onClick={(): void => onSort('created_at')}
						>
							{t('funnels.sort_last_created')}
							{sortOrder.columnKey === 'created_at' && <Check size={14} />}
						</Button>
						<Button
							type="text"
							className="sort-popover-content__button"
							onClick={(): void => onSort('updated_at')}
						>
							{t('funnels.sort_last_updated')}
							{sortOrder.columnKey === 'updated_at' && <Check size={14} />}
						</Button>
					</div>
				}
				rootClassName="sort-popover"
				placement="bottomRight"
				arrow={false}
			>
				<Button type="text" className="search__sort-btn">
					<ArrowDownWideNarrow size={12} data-testid="sort-by" />
					<div className="search__sort-btn-text">{t('funnels.sort')}</div>
				</Button>
			</Popover>
			<Input
				className="search__input"
				placeholder={t('search_funnels_placeholder')}
				prefix={
					<Search
						size={12}
						color={Color.BG_VANILLA_400}
						style={{ opacity: '0.4' }}
					/>
				}
				value={searchQuery}
				onChange={onSearch}
			/>
			<Tooltip
				title={
					!hasEditPermission
						? t('funnels.perm_create_funnels')
						: ''
				}
			>
				<Button
					type="primary"
					icon={<Plus size={16} />}
					className="search__new-btn"
					onClick={onCreateFunnel}
					disabled={!hasEditPermission}
				>
					{t('funnels.new_funnel')}
				</Button>
			</Tooltip>
		</div>
	);
}

export default SearchBar;
