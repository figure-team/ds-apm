import { fireEvent, render, screen } from 'tests/test-utils';
import type { SopDocumentSummary } from 'api/v2/rules/sopDocuments';

import SopDocumentDetail from '../SopDocumentDetail';

jest.mock('container/Runbooks/RunbooksSection', () => ({
	__esModule: true,
	default: (): JSX.Element => <div>RunbooksSectionStub</div>,
}));

const record: SopDocumentSummary = {
	approvalStatus: 'approved',
	checksum: 'sha256:existing',
	contractVersion: 'ds.sop_document.v1',
	displayUrl: 'https://kb.example/sop/SOP-PAY-001',
	ownerTeam: 'payments',
	sopId: 'SOP-PAY-001',
	source: { sourceId: 'src-managed-markdown-default', type: 'managed_markdown' },
	tags: ['payment-api'],
	tenantScope: { environments: ['prod'], projectIds: ['customer-a'] },
	title: 'Payment API 5xx response',
	updatedAt: '2026-05-12T00:00:00Z',
	version: '2026-05-12.1',
};

describe('SopDocumentDetail', () => {
	it('renders SOP meta, the runbooks body, and fires callbacks', () => {
		const onBack = jest.fn();
		const onEditDocument = jest.fn();
		render(
			<SopDocumentDetail
				record={record}
				onBack={onBack}
				onEditDocument={onEditDocument}
			/>,
		);

		expect(screen.getByText('Payment API 5xx response')).toBeInTheDocument();
		expect(screen.getByText('SOP-PAY-001')).toBeInTheDocument();
		expect(screen.getByText('RunbooksSectionStub')).toBeInTheDocument();

		fireEvent.click(screen.getByTestId('sop-detail-back'));
		expect(onBack).toHaveBeenCalledTimes(1);

		fireEvent.click(screen.getByTestId('sop-detail-edit-document'));
		expect(onEditDocument).toHaveBeenCalledWith(record);
	});
});
