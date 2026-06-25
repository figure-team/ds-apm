import { useTranslation } from 'react-i18next';
import { ArrowLeftOutlined, EditOutlined } from '@ant-design/icons';
import { Button, Tag } from 'antd';
import type { SopDocumentSummary } from 'api/v2/rules/sopDocuments';
import RunbooksSection from 'container/Runbooks/RunbooksSection';

interface SopDocumentDetailProps {
	record: SopDocumentSummary;
	onBack: () => void;
	onEditDocument: (record: SopDocumentSummary) => void;
}

export default function SopDocumentDetail({
	record,
	onBack,
	onEditDocument,
}: SopDocumentDetailProps): JSX.Element {
	const { t } = useTranslation(['sop_documents']);

	return (
		<div className="sop-documents-page__detail">
			<Button
				className="sop-documents-page__detail-back"
				data-testid="sop-detail-back"
				icon={<ArrowLeftOutlined />}
				onClick={onBack}
				type="link"
			>
				{t('btn_back_to_list')}
			</Button>

			<div className="sop-documents-page__detail-header">
				<div>
					<h1>{record.title}</h1>
					<div className="sop-documents-page__detail-meta">
						<span>{record.sopId}</span>
						<span>{record.version}</span>
						<span>{record.ownerTeam}</span>
						<Tag color={record.approvalStatus === 'approved' ? 'green' : 'default'}>
							{t(`status_${record.approvalStatus}`)}
						</Tag>
					</div>
					<p>{t('detail_runbooks_hint')}</p>
				</div>
				<Button
					data-testid="sop-detail-edit-document"
					icon={<EditOutlined />}
					onClick={(): void => onEditDocument(record)}
				>
					{t('btn_edit_document')}
				</Button>
			</div>

			<RunbooksSection sopId={record.sopId} version={record.version} />
		</div>
	);
}
