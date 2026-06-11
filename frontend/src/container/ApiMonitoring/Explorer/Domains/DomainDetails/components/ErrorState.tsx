import { Button, Typography } from 'antd';
import { RotateCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import awwSnapUrl from '@/assets/Icons/awwSnap.svg';

function ErrorState({ refetch }: { refetch: () => void }): JSX.Element {
	const { t } = useTranslation('apiMonitoring');
	return (
		<div className="error-state-container">
			<div className="error-state-content-wrapper">
				<div className="error-state-content">
					<div className="icon">
						<img src={awwSnapUrl} alt="awwSnap" width={32} height={32} />
					</div>
					<div className="error-state-text">
						<Typography.Text>{t('error_ran_into')}</Typography.Text>
						<Typography.Text type="secondary">
							{t('error_refresh_panel')}
						</Typography.Text>
					</div>
				</div>
				<Button
					className="refresh-cta"
					onClick={(): void => refetch()}
					icon={<RotateCw size={16} />}
				>
					{t('refresh_this_panel')}
				</Button>
			</div>
		</div>
	);
}

export default ErrorState;
