import { LinkOutlined } from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import { useCopySpanLink } from 'hooks/trace/useCopySpanLink';
import { useTranslation } from 'react-i18next';
import { Span } from 'types/api/trace/getTraceV2';

import './SpanLineActionButtons.styles.scss';

export interface SpanLineActionButtonsProps {
	span: Span;
}
export default function SpanLineActionButtons({
	span,
}: SpanLineActionButtonsProps): JSX.Element {
	const { onSpanCopy } = useCopySpanLink(span);
	const { t } = useTranslation(['trace']);

	return (
		<div className="span-line-action-buttons">
			<Tooltip title={t('copy_span_link')}>
				<Button
					size="small"
					icon={<LinkOutlined size={14} />}
					onClick={onSpanCopy}
					className="copy-span-btn"
				/>
			</Tooltip>
		</div>
	);
}
