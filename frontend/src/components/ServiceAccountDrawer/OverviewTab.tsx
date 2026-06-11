import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { LockKeyhole } from '@signozhq/icons';
import { Badge, Input } from '@signozhq/ui';
import type { AuthtypesRoleDTO } from 'api/generated/services/sigNoz.schemas';
import RolesSelect from 'components/RolesSelect';
import { DATE_TIME_FORMATS } from 'constants/dateTimeFormats';
import { ServiceAccountRow } from 'container/ServiceAccountsSettings/utils';
import { useTimezone } from 'providers/Timezone';
import APIError from 'types/api/error';

import SaveErrorItem from './SaveErrorItem';
import type { SaveError } from './utils';

interface OverviewTabProps {
	account: ServiceAccountRow;
	localName: string;
	onNameChange: (v: string) => void;
	localRole: string;
	onRoleChange: (v: string | undefined) => void;
	isDisabled: boolean;
	availableRoles: AuthtypesRoleDTO[];
	rolesLoading?: boolean;
	rolesError?: boolean;
	rolesErrorObj?: APIError | undefined;
	onRefetchRoles?: () => void;
	saveErrors?: SaveError[];
}

function OverviewTab({
	account,
	localName,
	onNameChange,
	localRole,
	onRoleChange,
	isDisabled,
	availableRoles,
	rolesLoading,
	rolesError,
	rolesErrorObj,
	onRefetchRoles,
	saveErrors = [],
}: OverviewTabProps): JSX.Element {
	const { t } = useTranslation('serviceAccounts');
	const { formatTimezoneAdjustedTimestamp } = useTimezone();

	const formatTimestamp = useCallback(
		(ts: string | null | undefined): string => {
			if (!ts) {
				return '—';
			}
			const d = new Date(ts);
			if (Number.isNaN(d.getTime())) {
				return '—';
			}
			return formatTimezoneAdjustedTimestamp(ts, DATE_TIME_FORMATS.DASH_DATETIME);
		},
		[formatTimezoneAdjustedTimestamp],
	);

	return (
		<>
			<div className="sa-drawer__field">
				<label className="sa-drawer__label" htmlFor="sa-name">
					{t('name')}
				</label>
				{isDisabled ? (
					<div className="sa-drawer__input-wrapper sa-drawer__input-wrapper--disabled">
						<span className="sa-drawer__input-text">{localName || '—'}</span>
						<LockKeyhole size={14} className="sa-drawer__lock-icon" />
					</div>
				) : (
					<Input
						id="sa-name"
						value={localName}
						onChange={(e): void => onNameChange(e.target.value)}
						className="sa-drawer__input"
						placeholder={t('name_placeholder')}
					/>
				)}
			</div>

			<div className="sa-drawer__field">
				<label className="sa-drawer__label" htmlFor="sa-email">
					{t('email_address')}
				</label>
				<div className="sa-drawer__input-wrapper sa-drawer__input-wrapper--disabled">
					<span className="sa-drawer__input-text">{account.email || '—'}</span>
					<LockKeyhole size={14} className="sa-drawer__lock-icon" />
				</div>
			</div>

			<div className="sa-drawer__field">
				<label className="sa-drawer__label" htmlFor="sa-roles">
					{t('roles')}
				</label>
				{isDisabled ? (
					<div className="sa-drawer__input-wrapper sa-drawer__input-wrapper--disabled">
						<div className="sa-drawer__disabled-roles">
							{localRole ? (
								<Badge color="vanilla">
									{availableRoles.find((r) => r.id === localRole)?.name ?? localRole}
								</Badge>
							) : (
								<span className="sa-drawer__input-text">—</span>
							)}
						</div>
						<LockKeyhole size={14} className="sa-drawer__lock-icon" />
					</div>
				) : (
					<RolesSelect
						id="sa-roles"
						roles={availableRoles}
						loading={rolesLoading}
						isError={rolesError}
						error={rolesErrorObj}
						onRefetch={onRefetchRoles}
						value={localRole}
						onChange={onRoleChange}
						placeholder={t('select_role_placeholder')}
					/>
				)}
			</div>

			<div className="sa-drawer__meta">
				<div className="sa-drawer__meta-item">
					<span className="sa-drawer__meta-label">{t('status')}</span>
					{account.status?.toUpperCase() === 'ACTIVE' ? (
						<Badge color="forest" variant="outline">
							{t('status_active')}
						</Badge>
					) : account.status?.toUpperCase() === 'DELETED' ? (
						<Badge color="cherry" variant="outline">
							{t('status_deleted')}
						</Badge>
					) : (
						<Badge color="vanilla" variant="outline" className="sa-status-badge">
							{account.status ? account.status.toUpperCase() : 'UNKNOWN'}
						</Badge>
					)}
				</div>

				<div className="sa-drawer__meta-item">
					<span className="sa-drawer__meta-label">{t('created_at')}</span>
					<Badge color="vanilla">{formatTimestamp(account.createdAt)}</Badge>
				</div>

				<div className="sa-drawer__meta-item">
					<span className="sa-drawer__meta-label">{t('updated_at')}</span>
					<Badge color="vanilla">{formatTimestamp(account.updatedAt)}</Badge>
				</div>
			</div>

			{saveErrors.length > 0 && (
				<div className="sa-drawer__save-errors">
					{saveErrors.map(({ context, apiError, onRetry }) => (
						<SaveErrorItem
							key={context}
							context={context}
							apiError={apiError}
							onRetry={onRetry}
						/>
					))}
				</div>
			)}
		</>
	);
}

export default OverviewTab;
