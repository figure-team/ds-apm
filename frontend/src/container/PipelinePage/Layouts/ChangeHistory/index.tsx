import { useTranslation } from 'react-i18next';
import { Table } from 'antd';
import { Pipeline } from 'types/api/pipeline/def';

import { getChangeHistoryColumns } from '../../PipelineListsView/config';
import { HistoryTableWrapper } from '../../styles';
import { historyPagination } from '../config';

function ChangeHistory({ pipelineData }: ChangeHistoryProps): JSX.Element {
	const { t } = useTranslation('pipeline');
	return (
		<HistoryTableWrapper>
			<Table
				columns={getChangeHistoryColumns(t)}
				dataSource={pipelineData?.history ?? []}
				rowKey="id"
				pagination={historyPagination}
			/>
		</HistoryTableWrapper>
	);
}

interface ChangeHistoryProps {
	pipelineData: Pipeline;
}

export default ChangeHistory;
