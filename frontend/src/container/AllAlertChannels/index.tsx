import { PlusOutlined } from '@ant-design/icons';
import { Tabs, Tooltip, Typography } from 'antd';
import getAll from 'api/channels/getAll';
import logEvent from 'api/common/logEvent';
import Spinner from 'components/Spinner';
import TextToolTip from 'components/TextToolTip';
import ROUTES from 'constants/routes';
import DLQFailures from 'container/DLQFailures';
import useComponentPermission from 'hooks/useComponentPermission';
import history from 'lib/history';
import { isUndefined } from 'lodash-es';
import { useAppContext } from 'providers/App/App';
import { useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from 'react-query';
import { SuccessResponseV2 } from 'types/api';
import { Channels } from 'types/api/channels/getAll';
import APIError from 'types/api/error';

import AlertChannelsComponent from './AlertChannels';
import { Button, ButtonContainer, RightActionContainer } from './styles';

import './AllAlertChannels.styles.scss';

const { Paragraph } = Typography;

function ChannelListTab(): JSX.Element {
	const { t } = useTranslation(['channels']);
	const { user } = useAppContext();
	const [addNewChannelPermission] = useComponentPermission(
		['add_new_channel'],
		user.role,
	);
	const onToggleHandler = useCallback(() => {
		history.push(ROUTES.CHANNELS_NEW);
	}, []);

	const { isLoading, data, error } = useQuery<
		SuccessResponseV2<Channels[]>,
		APIError
	>(['getChannels'], {
		queryFn: () => getAll(),
	});

	useEffect(() => {
		if (!isUndefined(data?.data)) {
			logEvent('Alert Channel: Channel list page visited', {
				number: data?.data?.length,
			});
		}
	}, [data?.data]);

	if (error) {
		return <Typography>{error.getErrorMessage()}</Typography>;
	}

	if (isLoading || isUndefined(data?.data)) {
		return <Spinner tip={t('loading_channels_message')} height="90vh" />;
	}

	return (
		<div className="alert-channels-container">
			<ButtonContainer>
				<Paragraph ellipsis type="secondary">
					{t('sending_channels_note')}
				</Paragraph>

				<RightActionContainer>
					<TextToolTip
						text={t('tooltip_notification_channels')}
						url="https://signoz.io/docs/userguide/alerts-management/#setting-notification-channel"
					/>

					<Tooltip
						title={
							!addNewChannelPermission
								? t('tooltip_admin_create_channel')
								: undefined
						}
					>
						<Button
							onClick={onToggleHandler}
							icon={<PlusOutlined />}
							disabled={!addNewChannelPermission}
						>
							{t('button_new_channel')}
						</Button>
					</Tooltip>
				</RightActionContainer>
			</ButtonContainer>

			<AlertChannelsComponent allChannels={data?.data || []} />
		</div>
	);
}

function AlertChannels(): JSX.Element {
	const { t } = useTranslation(['channels']);

	const tabItems = [
		{
			key: 'channels',
			label: t('tab_channel_list'),
			children: <ChannelListTab />,
		},
		{
			key: 'dlq',
			label: t('tab_dlq'),
			children: <DLQFailures />,
		},
	];

	return (
		<div className="settings-shell">
			<Tabs items={tabItems} defaultActiveKey="channels" />
		</div>
	);
}

export default AlertChannels;
