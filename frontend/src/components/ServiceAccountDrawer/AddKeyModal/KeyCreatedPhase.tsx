import { useTranslation } from 'react-i18next';
import { Check, Copy } from '@signozhq/icons';
import { Badge, Button, Callout } from '@signozhq/ui';
import type { ServiceaccounttypesGettableFactorAPIKeyWithKeyDTO } from 'api/generated/services/sigNoz.schemas';

export interface KeyCreatedPhaseProps {
	createdKey: ServiceaccounttypesGettableFactorAPIKeyWithKeyDTO;
	hasCopied: boolean;
	expiryLabel: string;
	onCopy: () => void;
}

function KeyCreatedPhase({
	createdKey,
	hasCopied,
	expiryLabel,
	onCopy,
}: KeyCreatedPhaseProps): JSX.Element {
	const { t } = useTranslation('serviceAccounts');
	return (
		<div className="add-key-modal__form">
			<div className="add-key-modal__field">
				<span className="add-key-modal__label">{t('key')}</span>
				<div className="add-key-modal__key-display">
					<span className="add-key-modal__key-text">{createdKey.key}</span>
					<Button
						variant="outlined"
						color="secondary"
						size="sm"
						onClick={onCopy}
						className="add-key-modal__copy-btn"
					>
						{hasCopied ? <Check size={12} /> : <Copy size={12} />}
					</Button>
				</div>
			</div>

			<div className="add-key-modal__expiry-meta">
				<span className="add-key-modal__expiry-label">{t('expiration')}</span>
				<Badge color="vanilla">{expiryLabel}</Badge>
			</div>

			<div className="add-key-modal__callout-wrapper">
				<Callout
					type="info"
					showIcon
					title={t('store_key_securely')}
				/>
			</div>
		</div>
	);
}

export default KeyCreatedPhase;
