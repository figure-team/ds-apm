import { CopyOutlined } from '@ant-design/icons';
import { Button, Drawer, Typography } from 'antd';

interface DLQPayloadDrawerProps {
	payload: string | null; // base64
	onClose: () => void;
}

export function DLQPayloadDrawer({
	payload,
	onClose,
}: DLQPayloadDrawerProps): JSX.Element {
	let decoded = '';
	if (payload) {
		try {
			const raw = atob(payload);
			decoded = JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			decoded = '(페이로드 파싱 실패)';
		}
	}

	return (
		<Drawer
			title="Alert Payload"
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
					복사
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
