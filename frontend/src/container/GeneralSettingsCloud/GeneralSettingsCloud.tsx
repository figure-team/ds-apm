import { Card, Typography } from 'antd';
import { Info } from 'lucide-react';
import { Trans, useTranslation } from 'react-i18next';

import './GeneralSettingsCloud.styles.scss';

export default function GeneralSettingsCloud(): JSX.Element {
	const { t } = useTranslation(['generalSettings']);
	return (
		<Card className="general-settings-container">
			<Info size={16} />
			<Typography.Text>
				<Trans
					t={t}
					i18nKey="cloud_retention_contact"
					components={[
						// eslint-disable-next-line jsx-a11y/anchor-has-content
						<a key="0" href="mailto:cloud-support@signoz.io">
							{' '}
						</a>,
					]}
				/>
			</Typography.Text>
		</Card>
	);
}
