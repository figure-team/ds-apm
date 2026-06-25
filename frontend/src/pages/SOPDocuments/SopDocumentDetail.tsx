import { Drawer } from 'antd';
import type { SopDocumentSummary } from 'api/v2/rules/sopDocuments';
import RunbooksSection from 'container/Runbooks/RunbooksSection';

interface SopDocumentDetailProps {
	open: boolean;
	record?: SopDocumentSummary;
	onClose: () => void;
}

export default function SopDocumentDetail({
	open,
	record,
	onClose,
}: SopDocumentDetailProps): JSX.Element {
	return (
		<Drawer
			className="sop-documents-page__detail-drawer"
			data-testid="sop-document-detail-drawer"
			destroyOnClose
			onClose={onClose}
			open={open}
			title={record?.title}
			width={820}
		>
			{record && (
				<RunbooksSection sopId={record.sopId} version={record.version} />
			)}
		</Drawer>
	);
}
