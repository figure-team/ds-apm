import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, toast } from '@signozhq/ui';
import { Tooltip } from 'antd';
import { convertToApiError } from 'api/ErrorResponseHandlerForGeneratedAPIs';
import type { RenderErrorResponseDTO } from 'api/generated/services/sigNoz.schemas';
import { AxiosError } from 'axios';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { Check, Loader, Send, X } from 'lucide-react';
import { useErrorModal } from 'providers/ErrorModalProvider';
import { toPostableRuleDTO } from 'types/api/alerts/convert';
import APIError from 'types/api/error';
import { isModifierKeyPressed } from 'utils/app';

import { useCreateAlertState } from '../context';
import {
	buildCreateThresholdAlertRulePayload,
	validateCreateAlertState,
} from './utils';

import './styles.scss';

function Footer(): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const {
		alertType,
		alertState: basicAlertState,
		thresholdState,
		advancedOptions,
		evaluationWindow,
		notificationSettings,
		discardAlertRule,
		createAlertRule,
		isCreatingAlertRule,
		testAlertRule,
		isTestingAlertRule,
		updateAlertRule,
		isUpdatingAlertRule,
		isEditMode,
		ruleId,
	} = useCreateAlertState();
	const { currentQuery } = useQueryBuilder();
	const { safeNavigate } = useSafeNavigate();
	const { showErrorModal } = useErrorModal();

	const handleApiError = useCallback(
		(error: unknown): void => {
			showErrorModal(
				convertToApiError(error as AxiosError<RenderErrorResponseDTO>) as APIError,
			);
		},
		[showErrorModal],
	);

	const handleDiscard = (e: React.MouseEvent): void => {
		discardAlertRule();
		safeNavigate('/alerts', { newTab: isModifierKeyPressed(e) });
	};

	const alertValidationMessage = useMemo(
		() =>
			validateCreateAlertState({
				alertType,
				basicAlertState,
				thresholdState,
				advancedOptions,
				evaluationWindow,
				notificationSettings,
				query: currentQuery,
			}),
		[
			alertType,
			basicAlertState,
			thresholdState,
			advancedOptions,
			evaluationWindow,
			notificationSettings,
			currentQuery,
		],
	);

	const handleTestNotification = useCallback((): void => {
		const payload = buildCreateThresholdAlertRulePayload({
			alertType,
			basicAlertState,
			thresholdState,
			advancedOptions,
			evaluationWindow,
			notificationSettings,
			query: currentQuery,
		});
		testAlertRule(
			{ data: toPostableRuleDTO(payload) },
			{
				onSuccess: (response) => {
					if (response.data?.alertCount === 0) {
						toast.error(t('no_alerts_found'));
						return;
					}
					toast.success(t('rule_test_fired'));
				},
				onError: handleApiError,
			},
		);
	}, [
		alertType,
		basicAlertState,
		thresholdState,
		advancedOptions,
		evaluationWindow,
		notificationSettings,
		currentQuery,
		testAlertRule,
	]);

	const handleSaveAlert = useCallback((): void => {
		const payload = buildCreateThresholdAlertRulePayload({
			alertType,
			basicAlertState,
			thresholdState,
			advancedOptions,
			evaluationWindow,
			notificationSettings,
			query: currentQuery,
		});
		if (isEditMode) {
			updateAlertRule(
				{
					pathParams: { id: ruleId },
					data: toPostableRuleDTO(payload),
				},
				{
					onSuccess: () => {
						toast.success(t('v2_alert_rule_updated'));
						safeNavigate('/alerts');
					},
					onError: handleApiError,
				},
			);
		} else {
			createAlertRule(
				{ data: toPostableRuleDTO(payload) },
				{
					onSuccess: () => {
						toast.success(t('v2_alert_rule_created'));
						safeNavigate('/alerts');
					},
					onError: handleApiError,
				},
			);
		}
	}, [
		alertType,
		basicAlertState,
		thresholdState,
		advancedOptions,
		evaluationWindow,
		notificationSettings,
		currentQuery,
		isEditMode,
		ruleId,
		updateAlertRule,
		createAlertRule,
		safeNavigate,
		handleApiError,
	]);

	const disableButtons =
		isCreatingAlertRule || isTestingAlertRule || isUpdatingAlertRule;

	const saveAlertButton = useMemo(() => {
		let button = (
			<Button
				variant="solid"
				color="primary"
				onClick={handleSaveAlert}
				disabled={disableButtons || Boolean(alertValidationMessage)}
			>
				{isCreatingAlertRule || isUpdatingAlertRule ? (
					<Loader size={14} />
				) : (
					<Check size={14} />
				)}
				{t('v2_save_alert_rule')}
			</Button>
		);
		if (alertValidationMessage) {
			button = <Tooltip title={t(alertValidationMessage)}>{button}</Tooltip>;
		}
		return button;
	}, [
		alertValidationMessage,
		disableButtons,
		handleSaveAlert,
		isCreatingAlertRule,
		isUpdatingAlertRule,
	]);

	const testAlertButton = useMemo(() => {
		let button = (
			<Button
				variant="solid"
				color="secondary"
				onClick={handleTestNotification}
				disabled={disableButtons || Boolean(alertValidationMessage)}
			>
				{isTestingAlertRule ? <Loader size={14} /> : <Send size={14} />}
				{t('v2_test_notification')}
			</Button>
		);
		if (alertValidationMessage) {
			button = <Tooltip title={t(alertValidationMessage)}>{button}</Tooltip>;
		}
		return button;
	}, [
		alertValidationMessage,
		disableButtons,
		handleTestNotification,
		isTestingAlertRule,
	]);

	return (
		<div className="create-alert-v2-footer">
			<Button
				variant="solid"
				color="secondary"
				onClick={handleDiscard}
				disabled={disableButtons}
			>
				<X size={14} /> {t('v2_discard')}
			</Button>
			<div className="button-group">
				<span
					className={`footer__validation-status ${
						alertValidationMessage
							? 'footer__validation-status--error'
							: 'footer__validation-status--ready'
					}`}
				>
					{alertValidationMessage ? t(alertValidationMessage) : t('v2_alert_ready_to_save')}
				</span>
				{testAlertButton}
				{saveAlertButton}
			</div>
		</div>
	);
}

export default Footer;
