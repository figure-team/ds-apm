import { useState } from 'react';
import { Alert, Modal, Upload } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';

type Props<TResult> = {
	open: boolean;
	title: string;
	/** 드래그 영역 아래 안내 문구. 도메인마다 "템플릿"/"양식" 용어가 달라 주입한다. */
	hint: string;
	parseFile: (file: File) => Promise<TResult>;
	onClose: () => void;
	onParsed: (result: TResult) => void;
};

/** .xlsx 파일 하나를 받아 도메인 파서에 넘기는 일괄 업로드 모달. */
function BulkUploadModal<TResult>({
	open,
	title,
	hint,
	parseFile,
	onClose,
	onParsed,
}: Props<TResult>): JSX.Element {
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
			const result = await parseFile(file);
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
			title={title}
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
				<p className="ant-upload-text">.xlsx 파일을 드래그하거나 클릭해서 선택</p>
				<p className="ant-upload-hint">{hint}</p>
			</Upload.Dragger>
		</Modal>
	);
}

export default BulkUploadModal;
