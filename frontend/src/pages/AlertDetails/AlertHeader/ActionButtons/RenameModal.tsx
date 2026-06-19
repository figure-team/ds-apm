import { useCallback, useEffect, useRef } from 'react';
import { Button, Input, InputRef, Modal, Typography } from 'antd';
import { Check, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import './RenameModal.styles.scss';

type Props = {
	isOpen: boolean;
	setIsOpen: (isOpen: boolean) => void;
	onNameChangeHandler: () => void;
	isLoading: boolean;
	intermediateName: string;
	setIntermediateName: (name: string) => void;
};

function RenameModal({
	isOpen,
	setIsOpen,
	onNameChangeHandler,
	isLoading,
	intermediateName,
	setIntermediateName,
}: Props): JSX.Element {
	const { t } = useTranslation('alerts');
	const inputRef = useRef<InputRef>(null);

	useEffect(() => {
		if (isOpen && inputRef.current) {
			inputRef.current.focus();
		}
	}, [isOpen]);

	const handleClose = useCallback((): void => setIsOpen(false), [setIsOpen]);

	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent): void => {
			if (isOpen) {
				if (e.key === 'Enter') {
					onNameChangeHandler();
				} else if (e.key === 'Escape') {
					handleClose();
				}
			}
		};

		document.addEventListener('keydown', handleKeyDown);

		return (): void => {
			document.removeEventListener('keydown', handleKeyDown);
		};
	}, [isOpen, onNameChangeHandler, handleClose]);

	return (
		<Modal
			open={isOpen}
			title={t('rename_alert_title')}
			onOk={onNameChangeHandler}
			onCancel={handleClose}
			rootClassName="rename-alert"
			footer={
				<div className="alert-rename">
					<Button
						type="primary"
						icon={<Check size={14} />}
						className="rename-btn"
						onClick={onNameChangeHandler}
						disabled={isLoading}
					>
						{t('rename_alert_btn')}
					</Button>
					<Button
						type="text"
						icon={<X size={14} />}
						className="cancel-btn"
						onClick={handleClose}
					>
						{t('rename_cancel')}
					</Button>
				</div>
			}
		>
			<div className="alert-content">
				<Typography.Text className="name-text">{t('rename_enter_name')}</Typography.Text>
				<Input
					ref={inputRef}
					data-testid="alert-name"
					className="alert-name-input"
					value={intermediateName}
					onChange={(e): void => setIntermediateName(e.target.value)}
				/>
			</div>
		</Modal>
	);
}

export default RenameModal;
