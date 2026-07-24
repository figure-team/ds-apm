import BulkUploadModal from 'components/BulkUpload/BulkUploadModal';
import { parseSopExcel, type ParseSopExcelResult } from './parseSopExcel';

type Props = {
	open: boolean;
	onClose: () => void;
	onParsed: (result: ParseSopExcelResult) => void;
};

function SopBulkUploadModal({ open, onClose, onParsed }: Props): JSX.Element {
	return (
		<BulkUploadModal<ParseSopExcelResult>
			hint="헤더 행 포함 Excel 파일. 템플릿을 먼저 다운로드하세요."
			onClose={onClose}
			onParsed={onParsed}
			open={open}
			parseFile={parseSopExcel}
			title="SOP 일괄 업로드"
		/>
	);
}

export default SopBulkUploadModal;
