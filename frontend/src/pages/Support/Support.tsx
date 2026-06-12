import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from 'react-query';
import { useHistory, useLocation } from 'react-router-dom';
import { Button, Card, Modal, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import updateCreditCardApi from 'api/v1/checkout/create';
import { FeatureKeys } from 'constants/features';
import { useNotifications } from 'hooks/useNotifications';
import { TFunction } from 'i18next';
import {
	ArrowUpRight,
	Book,
	CreditCard,
	Github,
	LifeBuoy,
	MessageSquare,
	Slack,
	X,
} from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { SuccessResponseV2 } from 'types/api';
import { CheckoutSuccessPayloadProps } from 'types/api/billing/checkout';
import APIError from 'types/api/error';
import { getBaseUrl } from 'utils/basePath';
import { openInNewTab } from 'utils/navigation';

import './Support.styles.scss';

const { Title, Text } = Typography;

interface Channel {
	key: any;
	name?: string;
	icon?: JSX.Element;
	title?: string;
	url: any;
	btnText?: string;
	isExternal?: boolean;
}

const channelsMap = {
	documentation: 'documentation',
	github: 'github',
	slack_community: 'slack_community',
	chat: 'chat',
	schedule_call: 'schedule_call',
	slack_connect: 'slack_connect',
};

const getSupportChannels = (t: TFunction): Channel[] => [
	{
		key: 'documentation',
		name: t('helpSupport:channel_documentation_name').toString(),
		icon: <Book size={16} />,
		title: t('helpSupport:channel_documentation_title').toString(),
		url: 'https://signoz.io/docs/',
		btnText: t('helpSupport:channel_documentation_btn').toString(),
		isExternal: true,
	},
	{
		key: 'github',
		name: t('helpSupport:channel_github_name').toString(),
		icon: <Github size={16} />,
		title: t('helpSupport:channel_github_title').toString(),
		url: 'https://github.com/SigNoz/signoz/issues',
		btnText: t('helpSupport:channel_github_btn').toString(),
		isExternal: true,
	},
	{
		key: 'slack_community',
		name: t('helpSupport:channel_slack_name').toString(),
		icon: <Slack size={16} />,
		title: t('helpSupport:channel_slack_title').toString(),
		url: 'https://signoz.io/slack',
		btnText: t('helpSupport:channel_slack_btn').toString(),
		isExternal: true,
	},
	{
		key: 'chat',
		name: t('helpSupport:channel_chat_name').toString(),
		icon: <MessageSquare size={16} />,
		title: t('helpSupport:channel_chat_title').toString(),
		url: '',
		btnText: t('helpSupport:channel_chat_btn').toString(),
		isExternal: false,
	},
];

export default function Support(): JSX.Element {
	const { t } = useTranslation(['helpSupport', 'common']);
	const history = useHistory();
	const { notifications } = useNotifications();
	const { trialInfo, featureFlags } = useAppContext();
	const [isAddCreditCardModalOpen, setIsAddCreditCardModalOpen] =
		useState(false);

	const supportChannels = getSupportChannels(t);

	const { pathname } = useLocation();
	const handleChannelWithRedirects = (url: string): void => {
		openInNewTab(url);
	};

	useEffect(() => {
		if (history?.location?.state) {
			const histroyState = history?.location?.state as any;

			if (histroyState && histroyState?.from) {
				logEvent(`Support : From URL : ${histroyState.from}`, {});
			}
		}

		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	const isPremiumChatSupportEnabled =
		featureFlags?.find((flag) => flag.name === FeatureKeys.PREMIUM_SUPPORT)
			?.active || false;

	const showAddCreditCardModal =
		!isPremiumChatSupportEnabled && !trialInfo?.trialConvertedToSubscription;

	const handleBillingOnSuccess = (
		data: SuccessResponseV2<CheckoutSuccessPayloadProps>,
	): void => {
		if (data?.data?.redirectURL) {
			const newTab = document.createElement('a');
			newTab.href = data.data.redirectURL;
			newTab.target = '_blank';
			newTab.rel = 'noopener noreferrer';
			newTab.click();
		}
	};

	const handleBillingOnError = (error: APIError): void => {
		notifications.error({
			message: error.getErrorCode(),
			description: error.getErrorMessage(),
		});
	};

	const { mutate: updateCreditCard, isLoading: isLoadingBilling } = useMutation(
		updateCreditCardApi,
		{
			onSuccess: (data) => {
				handleBillingOnSuccess(data);
			},
			onError: handleBillingOnError,
		},
	);

	const handleAddCreditCard = (): void => {
		logEvent('Add Credit card modal: Clicked', {
			source: `help & support`,
			page: pathname,
		});

		updateCreditCard({
			url: getBaseUrl(),
		});
	};

	const handleChat = (): void => {
		if (showAddCreditCardModal) {
			logEvent('Disabled Chat Support: Clicked', {
				source: `help & support`,
				page: pathname,
			});
			setIsAddCreditCardModalOpen(true);
		} else if (window.pylon) {
			window.Pylon('show');
		}
	};

	const handleChannelClick = (channel: Channel): void => {
		logEvent(`Support : ${channel.name}`, {});

		switch (channel.key) {
			case channelsMap.documentation:
			case channelsMap.github:
			case channelsMap.slack_community:
				handleChannelWithRedirects(channel.url);
				break;
			case channelsMap.chat:
				handleChat();
				break;
			default:
				handleChannelWithRedirects('https://signoz.io/slack');
				break;
		}
	};

	return (
		<div className="support-page-container">
			<header className="support-page-header">
				<div className="support-page-header-title" data-testid="support-page-title">
					<LifeBuoy size={16} />
					{t('helpSupport:page_title')}
				</div>
			</header>

			<div className="support-page-content">
				<div className="support-page-content-description">
					{t('helpSupport:page_description')}
				</div>

				<div className="support-channels">
					{supportChannels.map(
						(channel): JSX.Element => (
							<Card className="support-channel" key={channel.key}>
								<div className="support-channel-content">
									<Title ellipsis level={5} className="support-channel-title">
										{channel.icon}
										{channel.name}{' '}
									</Title>
									<Text> {channel.title} </Text>
								</div>

								<div className="support-channel-action">
									<Button
										className="periscope-btn secondary support-channel-btn"
										type="default"
										onClick={(): void => handleChannelClick(channel)}
									>
										<Text ellipsis>{channel.btnText} </Text>
										{channel.isExternal && <ArrowUpRight size={14} />}
									</Button>
								</div>
							</Card>
						),
					)}
				</div>
			</div>

			{/* Add Credit Card Modal */}
			<Modal
				className="add-credit-card-modal"
				title={
					<span className="title">{t('helpSupport:add_credit_card_title')}</span>
				}
				open={isAddCreditCardModalOpen}
				closable
				onCancel={(): void => setIsAddCreditCardModalOpen(false)}
				destroyOnClose
				footer={[
					<Button
						key="cancel"
						onClick={(): void => setIsAddCreditCardModalOpen(false)}
						className="cancel-btn"
						icon={<X size={16} />}
					>
						{t('common:cancel')}
					</Button>,
					<Button
						key="submit"
						type="primary"
						icon={<CreditCard size={16} />}
						size="middle"
						loading={isLoadingBilling}
						disabled={isLoadingBilling}
						onClick={handleAddCreditCard}
						className="add-credit-card-btn periscope-btn primary"
					>
						{t('helpSupport:add_credit_card_btn')}
					</Button>,
				]}
			>
				<Typography.Text className="add-credit-card-text">
					{t('helpSupport:credit_card_text_before')}
					<span className="highlight-text">{t('helpSupport:trial_plan')}</span>
					{t('helpSupport:credit_card_text_after')}
				</Typography.Text>
			</Modal>
		</div>
	);
}
