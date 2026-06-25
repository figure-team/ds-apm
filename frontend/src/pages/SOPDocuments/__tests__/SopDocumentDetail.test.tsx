import { render, screen } from 'tests/test-utils';
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
	it('renders the SOP title and the runbooks body in an open drawer', () => {
		render(<SopDocumentDetail open record={record} onClose={jest.fn()} />);

		// 제목은 드로어 타이틀로만 노출되고, 메타/안내 문구는 더 이상 렌더하지 않는다.
		expect(screen.getByText('Payment API 5xx response')).toBeInTheDocument();
		expect(screen.queryByText('SOP-PAY-001')).not.toBeInTheDocument();
		expect(screen.getByText('RunbooksSectionStub')).toBeInTheDocument();
	});

	it('does not render runbooks content when closed', () => {
		render(<SopDocumentDetail open={false} record={record} onClose={jest.fn()} />);

		// destroyOnClose: 드로어가 닫혀 있으면 Runbook 본문이 마운트되지 않는다.
		expect(screen.queryByText('RunbooksSectionStub')).not.toBeInTheDocument();
	});
});
