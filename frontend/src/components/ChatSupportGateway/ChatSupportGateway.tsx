import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from 'react-query';
import { useLocation } from 'react-router-dom';
import { Button, Modal, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import updateCreditCardApi from 'api/v1/checkout/create';
import { useNotifications } from 'hooks/useNotifications';
import { CreditCard, MessageSquareText, X } from 'lucide-react';
import { SuccessResponseV2 } from 'types/api';
import { CheckoutSuccessPayloadProps } from 'types/api/billing/checkout';
import APIError from 'types/api/error';
import { getBaseUrl } from 'utils/basePath';

export default function ChatSupportGateway(): JSX.Element {
	const { t } = useTranslation(['helpSupport', 'common']);
	const { notifications } = useNotifications();

	const [isAddCreditCardModalOpen, setIsAddCreditCardModalOpen] =
		useState(false);

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
	const { pathname } = useLocation();

	const handleAddCreditCard = (): void => {
		logEvent('Add Credit card modal: Clicked', {
			source: `chat support icon`,
			page: pathname,
		});

		updateCreditCard({
			url: getBaseUrl(),
		});
	};

	return (
		<>
			<div className="chat-support-gateway">
				<Button
					className="chat-support-gateway-btn"
					onClick={(): void => {
						logEvent('Disabled Chat Support: Clicked', {
							source: `chat support icon`,
							page: pathname,
						});

						setIsAddCreditCardModalOpen(true);
					}}
				>
					<MessageSquareText size={24} />
				</Button>
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
						className="add-credit-card-btn"
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
		</>
	);
}
