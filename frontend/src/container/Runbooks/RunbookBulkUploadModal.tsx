import BulkUploadModal from 'components/BulkUpload/BulkUploadModal';

import {
	parseRunbookExcel,
	type ParseRunbookExcelResult,
} from './parseRunbookExcel';

type Props = {
	open: boolean;
	onClose: () => void;
	onParsed: (result: ParseRunbookExcelResult) => void;
};

function RunbookBulkUploadModal({
	open,
	onClose,
	onParsed,
}: Props): JSX.Element {
	return (
		<BulkUploadModal<ParseRunbookExcelResult>
			hint="헤더 행 포함 Excel 파일. 양식을 먼저 다운로드하세요."
			onClose={onClose}
			onParsed={onParsed}
			open={open}
			parseFile={parseRunbookExcel}
			title="Runbook 일괄 업로드"
		/>
	);
}

export default RunbookBulkUploadModal;
