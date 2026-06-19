import React from 'react';
import { Button, Typography } from 'antd';
import ROUTES from 'constants/routes';
import { handleContactSupport } from 'container/Integrations/utils';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { LifeBuoy, List } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { isModifierKeyPressed } from 'utils/app';

import broomUrl from '@/assets/Icons/broom.svg';
import constructionUrl from '@/assets/Icons/construction.svg';
import noDataUrl from '@/assets/Icons/no-data.svg';

import './AlertNotFound.styles.scss';

interface AlertNotFoundProps {
	isTestAlert: boolean;
}

function AlertNotFound({ isTestAlert }: AlertNotFoundProps): JSX.Element {
	const { t } = useTranslation('alerts');
	const { isCloudUser: isCloudUserVal } = useGetTenantLicense();
	const { safeNavigate } = useSafeNavigate();

	const checkAllRulesHandler = (e: React.MouseEvent): void => {
		safeNavigate(ROUTES.LIST_ALL_ALERT, { newTab: isModifierKeyPressed(e) });
	};

	const contactSupportHandler = (): void => {
		handleContactSupport(isCloudUserVal);
	};

	return (
		<div className="alert-not-found">
			<section className="description">
				<img src={noDataUrl} alt="no-data" className="not-found-img" />
				<Typography.Text className="not-found-text">
					{t('not_found_message')}
				</Typography.Text>
				<Typography.Text className="not-found-text">
					{isTestAlert
						? 'This can happen in the following scenario -'
						: 'This can happen in either of the following scenarios -'}
				</Typography.Text>
			</section>
			<section className="reasons">
				{!isTestAlert && (
					<>
						<div className="reason">
							<img src={constructionUrl} alt="no-data" className="construction-img" />
							<Typography.Text className="text">
								{t('not_found_link_incorrect')}
							</Typography.Text>
						</div>
						<div className="reason">
							<img src={broomUrl} alt="no-data" className="broom-img" />
							<Typography.Text className="text">
								{t('not_found_deleted')}
							</Typography.Text>
						</div>
					</>
				)}
				{isTestAlert && (
					<div className="reason">
						<img src={broomUrl} alt="no-data" className="broom-img" />
						<Typography.Text className="text">
							{t('not_found_test_alert')}
						</Typography.Text>
					</div>
				)}
			</section>
			<section className="none-of-above">
				<Typography.Text className="text">
					{t('not_found_contact_support_msg')}
				</Typography.Text>
				<div className="action-btns">
					<Button
						className="action-btn"
						icon={<List size={14} />}
						onClick={checkAllRulesHandler}
					>
						{t('not_found_check_rules')}
					</Button>
					<Button
						className="action-btn"
						icon={<LifeBuoy size={14} />}
						onClick={contactSupportHandler}
					>
						{t('not_found_contact_support')}
					</Button>
				</div>
			</section>
		</div>
	);
}

export default AlertNotFound;
