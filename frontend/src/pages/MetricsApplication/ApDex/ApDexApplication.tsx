import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import { SettingOutlined } from '@ant-design/icons';
import { Popover } from 'antd';
import { IServiceName } from 'container/MetricsApplication/Tabs/types';
import { useGetApDexSettings } from 'hooks/apDex/useGetApDexSettings';
import { useNotifications } from 'hooks/useNotifications';

import { Button } from '../styles';
import ApDexSettings from './ApDexSettings';

function ApDexApplication(): JSX.Element {
	const { t } = useTranslation(['services']);
	const { servicename: encodedServiceName } = useParams<IServiceName>();
	const servicename = decodeURIComponent(encodedServiceName);

	const {
		data,
		isLoading,
		error,
		refetch: refetchGetApDexSetting,
	} = useGetApDexSettings(servicename);
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const { notifications } = useNotifications();

	useEffect(() => {
		if (error) {
			notifications.error({
				message: error.getErrorCode(),
				description: error.getErrorMessage(),
			});
		}
	}, [error, notifications]);

	const handlePopOverClose = (): void => {
		setIsOpen(false);
	};

	const handleOpenChange = (newOpen: boolean): void => {
		setIsOpen(newOpen);
	};

	return (
		<Popover
			placement="bottomRight"
			destroyTooltipOnHide
			trigger={['click']}
			showArrow={false}
			open={isOpen}
			onOpenChange={handleOpenChange}
			overlayClassName="ap-dex-settings-popover"
			content={
				<ApDexSettings
					servicename={servicename}
					handlePopOverClose={handlePopOverClose}
					isLoading={isLoading}
					data={data}
					refetchGetApDexSetting={refetchGetApDexSetting}
				/>
			}
		>
			<div className="ap-dex-settings-popover-content">
				<Button size="middle" icon={<SettingOutlined />}>
					{t('services:settings')}
				</Button>
			</div>
		</Popover>
	);
}

export default ApDexApplication;
