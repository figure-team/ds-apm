import { CopyOutlined } from '@ant-design/icons';
import { Button, Drawer, Typography } from 'antd';
import { useTranslation } from 'react-i18next';

interface DLQPayloadDrawerProps {
	payload: string | null; // base64
	onClose: () => void;
}

export function DLQPayloadDrawer({
	payload,
	onClose,
}: DLQPayloadDrawerProps): JSX.Element {
	const { t } = useTranslation(['channels']);

	let decoded = '';
	if (payload) {
		try {
			const raw = atob(payload);
			decoded = JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			decoded = t('dlq_payload_parse_failed');
		}
	}

	return (
		<Drawer
			title={t('dlq_drawer_title')}
			open={payload !== null}
			onClose={onClose}
			width={560}
			extra={
				<Button
					icon={<CopyOutlined />}
					onClick={(): void => {
						void navigator.clipboard.writeText(decoded);
					}}
				>
					{t('dlq_btn_copy')}
				</Button>
			}
		>
			<Typography.Text>
				<pre style={{ fontSize: 12, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
					{decoded}
				</pre>
			</Typography.Text>
		</Drawer>
	);
}
