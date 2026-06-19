import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { CircleAlert, CircleCheck, LoaderCircle } from '@signozhq/icons';
import { Button, DialogWrapper, Input } from '@signozhq/ui';
import { RenderErrorResponseDTO } from 'api/generated/services/sigNoz.schemas';
import { AxiosError } from 'axios';
import LaunchChatSupport from 'components/LaunchChatSupport/LaunchChatSupport';

interface CustomDomainEditModalProps {
	isOpen: boolean;
	onClose: () => void;
	customDomainSubdomain?: string;
	dnsSuffix: string;
	isLoading: boolean;
	updateDomainError: AxiosError<RenderErrorResponseDTO> | null;
	onClearError: () => void;
	onSubmit: (subdomain: string) => void;
}

// eslint-disable-next-line sonarjs/cognitive-complexity
export default function CustomDomainEditModal({
	isOpen,
	onClose,
	customDomainSubdomain,
	dnsSuffix,
	isLoading,
	updateDomainError,
	onClearError,
	onSubmit,
}: CustomDomainEditModalProps): JSX.Element {
	const { t } = useTranslation('customDomain');
	const initialSubdomain = customDomainSubdomain ?? '';
	const [value, setValue] = useState(initialSubdomain);
	const [validationError, setValidationError] = useState<string | null>(null);

	useEffect(() => {
		if (isOpen) {
			setValue(initialSubdomain);
		}
	}, [isOpen, initialSubdomain]);

	const handleClose = (): void => {
		setValidationError(null);
		onClearError();
		onClose();
	};

	const handleChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
		setValue(e.target.value);
		setValidationError(null);
		onClearError();
	};

	const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
		if (e.key === 'Enter') {
			handleSubmit();
		}
	};

	const handleSubmit = (): void => {
		if (value === initialSubdomain) {
			setValidationError('Input is unchanged');
			return;
		}

		if (!value) {
			setValidationError('This field is required');
			return;
		}
		if (value.length < 3) {
			setValidationError('Minimum 3 characters required');
			return;
		}
		onSubmit(value);
	};

	const is409 = updateDomainError?.status === 409;

	const apiErrorMessage =
		(updateDomainError?.response?.data as RenderErrorResponseDTO)?.error
			?.message ?? null;

	const errorMessage =
		validationError ??
		(is409
			? (apiErrorMessage ??
				"You've already updated the custom domain once today. Please contact support.")
			: apiErrorMessage);

	const hasError = Boolean(errorMessage);

	const statusIcon = ((): JSX.Element | null => {
		if (isLoading) {
			return (
				<LoaderCircle size={16} className="animate-spin edit-modal-status-icon" />
			);
		}

		if (hasError) {
			return <CircleAlert size={16} color={Color.BG_CHERRY_500} />;
		}

		return value && value.length >= 3 ? (
			<CircleCheck size={16} color={Color.BG_FOREST_500} />
		) : null;
	})();

	return (
		<DialogWrapper
			className="edit-workspace-modal"
			title={t('edit_modal_title')}
			open={isOpen}
			onOpenChange={(open: boolean): void => {
				if (!open) {
					handleClose();
				}
			}}
			width="base"
		>
			<div className="edit-workspace-modal-content">
				<p className="edit-modal-description">
					{t('description')}{' '}
					<a
						href="https://signoz.io/support"
						target="_blank"
						rel="noreferrer"
						className="edit-modal-link"
					>
						{t('contact_support')}
					</a>
				</p>

				<div className="edit-modal-field">
					<label
						htmlFor="workspace-url-input"
						className={`edit-modal-label${
							hasError ? ' edit-modal-label--error' : ''
						}`}
					>
						{t('workspace_url_label')}
					</label>

					<div
						className={`edit-modal-input-wrapper${
							hasError ? ' edit-modal-input-wrapper--error' : ''
						}`}
					>
						<div className="edit-modal-input-field">
							{statusIcon}
							<Input
								id="workspace-url-input"
								aria-describedby="workspace-url-helper"
								aria-invalid={hasError}
								value={value}
								onChange={handleChange}
								onKeyDown={handleKeyDown}
								autoFocus
							/>
						</div>
						<div className="edit-modal-input-suffix">{dnsSuffix}</div>
					</div>

					<span
						id="workspace-url-helper"
						className={`edit-modal-helper${
							hasError ? ' edit-modal-helper--error' : ''
						}`}
					>
						{hasError
							? errorMessage
							: "To help you easily explore SigNoz, we've selected a tenant sub domain name for you."}
					</span>
				</div>

				<div className="edit-modal-note">
					<span className="edit-modal-note-emoji">🚧</span>
					<span className="edit-modal-note-text">{t('note_text')}</span>
				</div>

				<div className="edit-modal-footer">
					{is409 ? (
						<LaunchChatSupport
							attributes={{ screen: 'Custom Domain Settings' }}
							eventName="Custom Domain Settings: Facing Issues Updating Custom Domain"
							message={t('help_message')}
							buttonText="Contact Support"
						/>
					) : (
						<Button
							variant="solid"
							size="md"
							color="primary"
							className="edit-modal-apply-btn"
							onClick={handleSubmit}
							disabled={isLoading || value === initialSubdomain}
							loading={isLoading}
						>
							{t('apply_changes')}
						</Button>
					)}
				</div>
			</div>
		</DialogWrapper>
	);
}
