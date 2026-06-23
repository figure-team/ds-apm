import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Collapse } from 'antd';
import { Button, Input } from '@signozhq/ui';
import logEvent from 'api/common/logEvent';
import { previewSop, type PreviewSopResult } from 'api/v2/rules/previewSop';
import {
	listSopDocuments,
	type SopDocumentSummary,
} from 'api/v2/rules/sopDocuments';
import classNames from 'classnames';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';
import { RotateCcw } from 'lucide-react';
import type { Labels } from 'types/api/alerts/def';

import { useCreateAlertState } from '../context';
import { syncLabelsToExpression } from '../syncedLabels';
import {
	EVIDENCE_METADATA_FIELDS,
	validateEvidenceMetadata,
} from './evidenceMetadata';
import LabelsInput from './LabelsInput';
import {
	getMissingOperationalLabels,
	hasOperationalLabel,
	RECOMMENDED_OPERATIONAL_LABELS,
} from './operationalMetadata';
import {
	PM_BRIEFING_FIELDS,
	validatePmBriefingMetadata,
} from './pmBriefingMetadata';
import {
	getSopBindingStatus,
	hasSopBinding,
	resolveSopBindingDocument,
	SOP_ANNOTATION_FIELDS,
	SOP_ID_LABEL,
	validateSopAnnotations,
	validateSopLabelValue,
} from './sopMetadata';
import './styles.scss';

function CreateAlertHeader(): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { alertState, setAlertState, isEditMode } = useCreateAlertState();
	const [sopPreview, setSopPreview] = useState<PreviewSopResult>();
	const [sopPreviewError, setSopPreviewError] = useState('');
	const [isSopPreviewLoading, setIsSopPreviewLoading] = useState(false);
	const [sopDocuments, setSopDocuments] = useState<SopDocumentSummary[]>([]);

	useEffect(() => {
		listSopDocuments()
			.then((res) => setSopDocuments(res.data.documents))
			.catch(() => {});
	}, []);

	const { currentQuery, handleSetQueryData } = useQueryBuilder();
	const { safeNavigate } = useSafeNavigate();

	// labels -> query direction: mirror managed resource-attribute labels
	// (service.name, ...) back into the first builder query's filter expression.
	const handleLabelsChange = useCallback(
		(labels: Labels): void => {
			const firstQuery = currentQuery.builder.queryData?.[0];
			if (firstQuery) {
				const expression = firstQuery.filter?.expression || '';
				const nextExpression = syncLabelsToExpression(expression, labels);
				if (nextExpression !== expression) {
					handleSetQueryData(0, {
						...firstQuery,
						filter: { ...firstQuery.filter, expression: nextExpression },
					});
				}
			}
			setAlertState({ type: 'SET_ALERT_LABELS', payload: labels });
		},
		[currentQuery.builder.queryData, handleSetQueryData, setAlertState],
	);
	const urlQuery = useUrlQuery();

	const groupByLabels = useMemo(() => {
		const labels = new Array<string>();
		currentQuery.builder.queryData.forEach((query) => {
			query.groupBy.forEach((groupBy) => {
				labels.push(groupBy.key);
			});
		});
		return labels;
	}, [currentQuery]);

	// If the label key is a group by label, then it is not allowed to be used as a label key
	const validateLabelsKey = useCallback(
		(key: string): string | null => {
			if (groupByLabels.includes(key)) {
				return `Cannot use ${key} as a key`;
			}
			return null;
		},
		[groupByLabels],
	);

	const handleSwitchToClassicExperience = useCallback(() => {
		void logEvent('Alert: Switch to classic experience button clicked', {});

		urlQuery.set(QueryParams.showClassicCreateAlertsPage, 'true');
		const url = `${ROUTES.ALERTS_NEW}?${urlQuery.toString()}`;
		safeNavigate(url, { replace: true });
	}, [safeNavigate, urlQuery]);

	const handleAnnotationChange = useCallback(
		(key: string, value: string): void => {
			const nextAnnotations = {
				...alertState.annotations,
			};

			if (value.trim()) {
				nextAnnotations[key] = value;
			} else {
				delete nextAnnotations[key];
			}

			setSopPreview(undefined);
			setSopPreviewError('');
			setAlertState({
				type: 'SET_ALERT_ANNOTATIONS',
				payload: nextAnnotations,
			});
		},
		[alertState.annotations, setAlertState],
	);

	const handleLabelChange = useCallback(
		(key: string, value: string): void => {
			const nextLabels = {
				...alertState.labels,
			};

			if (value.trim()) {
				nextLabels[key] = value;
			} else {
				delete nextLabels[key];
			}

			setSopPreview(undefined);
			setSopPreviewError('');
			setAlertState({
				type: 'SET_ALERT_LABELS',
				payload: nextLabels,
			});
		},
		[alertState.labels, setAlertState],
	);

	const handleSopIdChange = useCallback(
		(value: string): void => {
			const trimmed = value.trim();
			const match = resolveSopBindingDocument(sopDocuments, trimmed);

			const nextLabels = { ...alertState.labels };
			if (trimmed) {
				nextLabels[SOP_ID_LABEL] = value;
			} else {
				delete nextLabels[SOP_ID_LABEL];
			}

			if (match) {
				if (match.ownerTeam) nextLabels.owner_team = match.ownerTeam;
				if (match.tenantScope.environments.length === 1) {
					nextLabels.environment = match.tenantScope.environments[0];
				}
				if (match.tenantScope.projectIds.length === 1) {
					nextLabels.project_id = match.tenantScope.projectIds[0];
				}
			}

			setSopPreview(undefined);
			setSopPreviewError('');
			setAlertState({
				type: 'SET_ALERT_LABELS',
				payload: nextLabels,
			});

			if (!match) return;

			const filled: Record<string, string> = {};
			if (match.displayUrl) filled.sop_url = match.displayUrl;
			filled.sop_source = match.source.sourceId;
			filled.sop_title = match.title;
			filled.sop_version = match.version;

			setAlertState({
				type: 'SET_ALERT_ANNOTATIONS',
				payload: { ...alertState.annotations, ...filled },
			});
		},
		[sopDocuments, alertState.labels, alertState.annotations, setAlertState],
	);

	const handlePreviewSop = useCallback(async (): Promise<void> => {
		setSopPreviewError('');
		setIsSopPreviewLoading(true);

		try {
			const response = await previewSop({
				labels: alertState.labels,
				annotations: alertState.annotations,
			});

			setSopPreview(response.data);
		} catch {
			setSopPreview(undefined);
			setSopPreviewError(t('v2_sop_preview_error'));
		} finally {
			setIsSopPreviewLoading(false);
		}
	}, [alertState.annotations, alertState.labels]);

	const pmBriefingWarnings = useMemo(
		() => validatePmBriefingMetadata(alertState.annotations),
		[alertState.annotations],
	);

	const evidenceMetadataWarnings = useMemo(
		() => validateEvidenceMetadata(alertState.annotations),
		[alertState.annotations],
	);

	const sopLabelWarnings = useMemo(
		() => validateSopLabelValue(alertState.labels[SOP_ID_LABEL]),
		[alertState.labels],
	);

	const sopAnnotationWarnings = useMemo(
		() => validateSopAnnotations(alertState.annotations),
		[alertState.annotations],
	);

	const missingOperationalLabels = useMemo(
		() => getMissingOperationalLabels(alertState.labels),
		[alertState.labels],
	);

	return (
		<div
			className={classNames('alert-header', { 'edit-alert-header': isEditMode })}
		>
			{!isEditMode && (
				<div className="alert-header__tab-bar">
					<div className="alert-header__tab">{t('v2_new_alert_rule')}</div>
					<Button
						prefix={<RotateCcw size={12} />}
						onClick={handleSwitchToClassicExperience}
						variant="solid"
						color="secondary"
						size="sm"
					>
						{t('v2_switch_to_classic')}
					</Button>
				</div>
			)}
			<div className="alert-header__content">
				<div className="alert-header__field-group">
					<label className="alert-header__field-label">
						{t('field_alert_name')}
						<span className="alert-header__required-badge">{t('v2_alert_name_required')}</span>
					</label>
					<Input
						type="text"
						value={alertState.name}
						onChange={(e): void =>
							setAlertState({ type: 'SET_ALERT_NAME', payload: e.target.value })
						}
						className="alert-header__input title"
						placeholder={t('v2_alert_name_placeholder')}
						data-testid="alert-name-input"
					/>
				</div>
				<div
					className="operational-metadata"
					aria-label={t('v2_sisam_routing_title')}
				>
					<div className="operational-metadata__header">
						<div className="operational-metadata__title">{t('v2_sisam_routing_title')}</div>
						<div className="operational-metadata__description">
							{t('v2_sisam_routing_desc')}
						</div>
					</div>
					<div
						className={classNames('operational-metadata__status', {
							'operational-metadata__status--complete':
								missingOperationalLabels.length === 0,
						})}
						role="status"
					>
						{missingOperationalLabels.length
							? t('v2_missing_labels', { count: missingOperationalLabels.length })
							: t('v2_all_labels_present')}
					</div>
					<div className="operational-metadata__labels">
						{RECOMMENDED_OPERATIONAL_LABELS.map(({ description, key, label }) => {
							const isPresent = hasOperationalLabel(alertState.labels, key);

							return (
								<div
									className={classNames('operational-metadata__label', {
										'operational-metadata__label--present': isPresent,
									})}
									key={key}
								>
									<span className="operational-metadata__label-name">
										{label}
										<span className="operational-metadata__label-key">{key}</span>
									</span>
									<span className="operational-metadata__label-description">{description}</span>
									<span className="operational-metadata__label-state">
										{isPresent ? t('v2_label_set') : t('v2_label_missing')}
									</span>
								</div>
							);
						})}
					</div>
				</div>
				<LabelsInput
					labels={alertState.labels}
					onLabelsChange={handleLabelsChange}
					validateLabelsKey={validateLabelsKey}
				/>
				<div className="sop-metadata" aria-label={t('v2_sop_binding_title')}>
					<div className="sop-metadata__header">
						<div className="sop-metadata__title">{t('v2_sop_binding_title')}</div>
						<div className="sop-metadata__description">
							{t('v2_sop_binding_desc')}
						</div>
					</div>
					<div
						className={classNames('sop-metadata__status', {
							'sop-metadata__status--complete': hasSopBinding(
								alertState.labels,
								alertState.annotations,
							),
						})}
						role="status"
					>
						{getSopBindingStatus(alertState.labels, alertState.annotations)}
					</div>
					{!hasSopBinding(alertState.labels, alertState.annotations) && (
						<div className="sop-metadata__missing-banner" role="alert">
							{t('v2_sop_missing_banner')} <strong>{t('v2_sop_id_field')}</strong> 또는 <strong>{t('v2_sop_url_field')}</strong>을 입력하세요.
						</div>
					)}
					<div className="sop-metadata__grid">
						<label className="sop-metadata__field">
							<span className="sop-metadata__label" style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
								{t('v2_sop_id_field')}
								<span className="alert-header__required-badge">{t('v2_alert_name_required')}</span>
							</span>
							<Input
								aria-describedby={
									sopLabelWarnings.length ? 'sop-metadata-sop-id-warning' : undefined
								}
								aria-invalid={sopLabelWarnings.length > 0}
								className="sop-metadata__input"
								data-testid="sop-metadata-sop_id"
								onChange={(event): void =>
									handleSopIdChange(event.target.value)
								}
								placeholder="SOP-PAY-001"
								type="text"
								value={alertState.labels[SOP_ID_LABEL] || ''}
							/>
							{sopLabelWarnings.map((warning, index) => (
								<span
									className="sop-metadata__warning"
									id={index === 0 ? 'sop-metadata-sop-id-warning' : undefined}
									key={warning}
									role="alert"
								>
									{warning}
								</span>
							))}
						</label>
						{SOP_ANNOTATION_FIELDS.slice(0, 2).map(({ key, label, placeholder }) => {
							const warnings = sopAnnotationWarnings[key] || [];
							const warningId = warnings.length
								? `sop-metadata-${key}-warning`
								: undefined;

							return (
								<label className="sop-metadata__field" key={key}>
									<span className="sop-metadata__label">{label}</span>
									<Input
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="sop-metadata__input"
										data-testid={`sop-metadata-${key}`}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										type="text"
										value={alertState.annotations[key] || ''}
									/>
									{warnings.map((warning, index) => (
										<span
											className="sop-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
						<label className="sop-metadata__field">
							<span className="sop-metadata__label">{t('v2_sop_project_id_field')}</span>
							<Input
								className="sop-metadata__input"
								data-testid="sop-metadata-project_id"
								onChange={(event): void =>
									handleLabelChange('project_id', event.target.value)
								}
								placeholder="customer-a"
								type="text"
								value={alertState.labels['project_id'] || ''}
							/>
						</label>
						<label className="sop-metadata__field">
							<span className="sop-metadata__label">{t('v2_sop_owner_team_field')}</span>
							<Input
								className="sop-metadata__input"
								data-testid="sop-metadata-owner_team"
								onChange={(event): void =>
									handleLabelChange('owner_team', event.target.value)
								}
								placeholder="payments-team"
								type="text"
								value={alertState.labels['owner_team'] || ''}
							/>
						</label>
						<label className="sop-metadata__field">
							<span className="sop-metadata__label">{t('v2_sop_environment_field')}</span>
							<Input
								className="sop-metadata__input"
								data-testid="sop-metadata-environment"
								onChange={(event): void =>
									handleLabelChange('environment', event.target.value)
								}
								placeholder={t('v2_sop_environment_placeholder')}
								type="text"
								value={alertState.labels['environment'] || ''}
							/>
						</label>
						{SOP_ANNOTATION_FIELDS.slice(2).map(({ key, label, placeholder }) => {
							const warnings = sopAnnotationWarnings[key] || [];
							const warningId = warnings.length
								? `sop-metadata-${key}-warning`
								: undefined;

							return (
								<label className="sop-metadata__field" key={key}>
									<span className="sop-metadata__label">{label}</span>
									<Input
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="sop-metadata__input"
										data-testid={`sop-metadata-${key}`}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										type="text"
										value={alertState.annotations[key] || ''}
									/>
									{warnings.map((warning, index) => (
										<span
											className="sop-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
					</div>
					<div className="sop-metadata__preview">
						<div className="sop-metadata__preview-header">
							<div>
								<div className="sop-metadata__preview-title">
									{t('v2_sop_preview_title')}
								</div>
								<div className="sop-metadata__preview-description">
									{t('v2_sop_preview_desc')}
								</div>
							</div>
							<Button
								color="secondary"
								disabled={isSopPreviewLoading}
								onClick={handlePreviewSop}
								size="sm"
								variant="solid"
							>
								{isSopPreviewLoading ? t('v2_previewing_btn') : t('v2_preview_sop_source_btn')}
							</Button>
						</div>
						{sopPreviewError && (
							<div className="sop-metadata__warning" role="alert">
								{sopPreviewError}
							</div>
						)}
						{sopPreview && (
							<div
								className="sop-metadata__preview-grid"
								data-testid="sop-source-preview"
							>
								<div className="sop-metadata__preview-summary">
									<span className="sop-metadata__preview-summary-title">
										{t('v2_review_summary')}
									</span>
									<span className="sop-metadata__preview-summary-copy">
										{sopPreview.status === 'bound'
											? t('v2_sop_ready_for_review')
											: t('v2_sop_add_metadata')}
									</span>
									<div className="sop-metadata__preview-badges">
										<span className="sop-metadata__preview-badge">
											{sopPreview.access.browserCredentialsAllowed
												? t('v2_browser_creds_allowed')
												: t('v2_browser_creds_blocked')}
										</span>
										{sopPreview.access.requiresServerSideFetch && (
											<span className="sop-metadata__preview-badge">
												{t('v2_server_side_connector_required')}
											</span>
										)}
										{sopPreview.access.auditEventRequired && (
											<span className="sop-metadata__preview-badge">
												{t('v2_audit_required')}
											</span>
										)}
									</div>
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_contract')}</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.contractVersion}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_status')}</span>
									<span
										className={classNames('sop-metadata__preview-value', {
											'sop-metadata__preview-value--complete':
												sopPreview.status === 'bound',
											'sop-metadata__preview-value--warning':
												sopPreview.status !== 'bound',
										})}
									>
										{sopPreview.status}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_source')}</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.source.name}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_search')}</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.search.query || t('v2_no_search_terms')}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_label')}</span>
									{sopPreview.preview.available && sopPreview.preview.url ? (
										<a
											className="sop-metadata__preview-link"
											href={sopPreview.preview.url}
											target="_blank"
											rel="noopener noreferrer"
										>
											{sopPreview.preview.displayUrl || sopPreview.preview.title}
										</a>
									) : (
										<span className="sop-metadata__preview-value">
											{sopPreview.preview.title || t('v2_preview_unavailable')}
										</span>
									)}
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_auth_boundary')}</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.mode} · {sopPreview.access.credentialScope}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">{t('v2_preview_service_account')}</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.recommendedServiceAccountProfile || t('v2_not_required')}
									</span>
								</div>
								<div className="sop-metadata__preview-note">
									<span className="sop-metadata__preview-label">
										{t('v2_preview_browser_creds')}
									</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.browserCredentialsAllowed
											? t('v2_creds_allowed')
											: t('v2_creds_never_accepted')}
									</span>
								</div>
								<div className="sop-metadata__preview-note">
									<span className="sop-metadata__preview-label">{t('v2_preview_boundary_note')}</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.message}
									</span>
								</div>
								{sopPreview.warnings?.map((warning) => (
									<div
										className="sop-metadata__preview-warning"
										key={warning}
										role="alert"
									>
										{warning}
									</div>
								))}
							</div>
						)}
					</div>
				</div>
				<Collapse ghost className="optional-sections-collapse">
				<Collapse.Panel header={t('v2_pm_briefing_collapse')} key="pm-briefing">
				<div
					className="pm-briefing-metadata"
					aria-label={t('v2_pm_briefing_collapse')}
				>
					<div className="pm-briefing-metadata__header">
						<div className="pm-briefing-metadata__description">
							{t('v2_pm_briefing_desc')}
						</div>
					</div>
					<div className="pm-briefing-metadata__grid">
						{PM_BRIEFING_FIELDS.map(({ key, label, placeholder }) => {
							const warnings = pmBriefingWarnings[key] || [];
							const warningId = warnings.length
								? `pm-briefing-${key}-warning`
								: undefined;

							return (
								<label className="pm-briefing-metadata__field" key={key}>
									<span className="pm-briefing-metadata__label">{label}</span>
									<textarea
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="pm-briefing-metadata__textarea"
										value={alertState.annotations[key] || ''}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										data-testid={`pm-briefing-${key}`}
										rows={2}
									/>
									{warnings.map((warning, index) => (
										<span
											className="pm-briefing-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
					</div>
				</div>
				</Collapse.Panel>
				<Collapse.Panel header={t('v2_ai_evidence_collapse')} key="ai-evidence">
				<div className="evidence-metadata" aria-label={t('v2_ai_evidence_collapse')}>
					<div className="evidence-metadata__header">
						<div className="evidence-metadata__description">
							{t('v2_ai_evidence_desc')}
						</div>
					</div>
					<div className="evidence-metadata__grid">
						{EVIDENCE_METADATA_FIELDS.map(({ key, label, placeholder }) => {
							const warnings = evidenceMetadataWarnings[key] || [];
							const warningId = warnings.length
								? `evidence-metadata-${key}-warning`
								: undefined;

							return (
								<label className="evidence-metadata__field" key={key}>
									<span className="evidence-metadata__label">{label}</span>
									<Input
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="evidence-metadata__input"
										data-testid={`evidence-metadata-${key}`}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										type="text"
										value={alertState.annotations[key] || ''}
									/>
									{warnings.map((warning, index) => (
										<span
											className="evidence-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
					</div>
				</div>
				</Collapse.Panel>
				</Collapse>
			</div>
		</div>
	);
}

export default CreateAlertHeader;
