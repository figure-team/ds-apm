import { useTranslation } from 'react-i18next';
import ReactDragListView from 'react-drag-listview';
import { TableComponents } from 'react-virtuoso';
import Spinner from 'components/Spinner';
import { dragColumnParams } from 'hooks/useDragColumns/configs';

import { TableStyled } from './styles';

interface LogsCustomTableProps {
	isLoading?: boolean;
	handleDragEnd: (fromIndex: number, toIndex: number) => void;
}

export const LogsCustomTable = ({
	isLoading,
	handleDragEnd,
}: LogsCustomTableProps): TableComponents['Table'] =>
	function CustomTable({ style, children }): JSX.Element {
		// eslint-disable-next-line react-hooks/rules-of-hooks
		const { t } = useTranslation(['logs']);
		if (isLoading) {
			return <Spinner height="35px" tip={t('logs:getting_logs')} />;
		}
		return (
			<ReactDragListView.DragColumn
				{...dragColumnParams}
				onDragEnd={handleDragEnd}
			>
				<TableStyled style={style}>{children}</TableStyled>
			</ReactDragListView.DragColumn>
		);
	};
