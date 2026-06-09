import { useState } from 'react';
import { Alert, Modal, Upload } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';
import { parseSopExcel, type ParseSopExcelResult } from './parseSopExcel';

type Props = {
	open: boolean;
	onClose: () => void;
	onParsed: (result: ParseSopExcelResult) => void;
};

function SopBulkUploadModal({ open, onClose, onParsed }: Props): JSX.Element {
	const [parsing, setParsing] = useState(false);
	const [parseError, setParseError] = useState('');

	const handleFile = async (file: File): Promise<false> => {
		if (!file.name.endsWith('.xlsx')) {
			setParseError('.xlsx 파일만 지원합니다.');
			return false;
		}
		setParsing(true);
		setParseError('');
		try {
			const result = await parseSopExcel(file);
			onParsed(result);
		} catch (err) {
			setParseError(err instanceof Error ? err.message : '파일 파싱 실패');
		} finally {
			setParsing(false);
		}
		return false; // prevent antd auto-upload
	};

	const handleClose = (): void => {
		setParseError('');
		onClose();
	};

	return (
		<Modal
			confirmLoading={parsing}
			footer={null}
			onCancel={handleClose}
			open={open}
			title="SOP 일괄 업로드"
			width={480}
		>
			{parseError && (
				<Alert
					message={parseError}
					showIcon
					style={{ marginBottom: 16 }}
					type="error"
				/>
			)}
			<Upload.Dragger
				accept=".xlsx"
				beforeUpload={handleFile}
				fileList={[] as UploadFile[]}
				multiple={false}
				showUploadList={false}
			>
				<p className="ant-upload-drag-icon">
					<InboxOutlined />
				</p>
				<p className="ant-upload-text">
					.xlsx 파일을 드래그하거나 클릭해서 선택
				</p>
				<p className="ant-upload-hint">
					헤더 행 포함 Excel 파일. 템플릿을 먼저 다운로드하세요.
				</p>
			</Upload.Dragger>
		</Modal>
	);
}

export default SopBulkUploadModal;
