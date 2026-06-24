import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@signozhq/ui';
import logEvent from 'api/common/logEvent';
import {
	getSopDocument,
	listSopDocuments,
	type SopDocument,
	type SopDocumentSummary,
} from 'api/v2/rules/sopDocuments';
import classNames from 'classnames';
import { MarkdownRenderer } from 'components/MarkdownRenderer/MarkdownRenderer';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';
import { RotateCcw } from 'lucide-react';
import type { Labels } from 'types/api/alerts/def';

import { useCreateAlertState } from '../context';
import { syncLabelsToExpression } from '../syncedLabels';
import LabelsInput from './LabelsInput';
import {
	getMissingOperationalLabels,
	hasOperationalLabel,
	RECOMMENDED_OPERATIONAL_LABELS,
} from './operationalMetadata';
import {
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
	const [isSopPreviewOpen, setIsSopPreviewOpen] = useState(false);
	const [sopDoc, setSopDoc] = useState<SopDocument>();
	const [sopDocError, setSopDocError] = useState('');
	const [isSopDocLoading, setIsSopDocLoading] = useState(false);
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

	const boundSopDocument = useMemo(
		() =>
			resolveSopBindingDocument(
				sopDocuments,
				alertState.labels[SOP_ID_LABEL] || '',
			),
		[sopDocuments, alertState.labels],
	);

	const handleToggleSopPreview = useCallback((): void => {
		setIsSopPreviewOpen((open) => !open);
	}, []);

	useEffect(() => {
		if (!isSopPreviewOpen || !boundSopDocument) {
			return undefined;
		}

		if (
			sopDoc?.sopId === boundSopDocument.sopId &&
			sopDoc?.version === boundSopDocument.version
		) {
			return undefined;
		}

		let cancelled = false;
		setIsSopDocLoading(true);
		setSopDocError('');
		getSopDocument(boundSopDocument.sopId, boundSopDocument.version)
			.then((res) => {
				if (!cancelled) {
					setSopDoc(res.data);
				}
			})
			.catch(() => {
				if (!cancelled) {
					setSopDoc(undefined);
					setSopDocError(t('v2_sop_doc_error'));
				}
			})
			.finally(() => {
				if (!cancelled) {
					setIsSopDocLoading(false);
				}
			});

		return (): void => {
			cancelled = true;
		};
	}, [isSopPreviewOpen, boundSopDocument, sopDoc, t]);

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
						{hasSopBinding(alertState.labels, alertState.annotations)
							? t('v2_sop_binding_present')
							: t('v2_sop_binding_missing')}
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
							<div className="sop-metadata__preview-title">
								{t('v2_sop_doc_preview_toggle')}
							</div>
							<Button
								color="secondary"
								onClick={handleToggleSopPreview}
								size="sm"
								variant="solid"
							>
								{isSopPreviewOpen
									? t('v2_sop_preview_collapse')
									: t('v2_sop_preview_expand')}
							</Button>
						</div>
						{isSopPreviewOpen && (
							<div
								className="sop-metadata__doc-preview"
								data-testid="sop-doc-preview"
							>
								{!boundSopDocument && (
									<div className="sop-metadata__doc-note">
										{t('v2_sop_doc_not_found')}
									</div>
								)}
								{boundSopDocument && isSopDocLoading && (
									<div className="sop-metadata__doc-note">
										{t('v2_sop_doc_loading')}
									</div>
								)}
								{boundSopDocument && !isSopDocLoading && sopDocError && (
									<div className="sop-metadata__warning" role="alert">
										{sopDocError}
									</div>
								)}
								{boundSopDocument &&
									!isSopDocLoading &&
									!sopDocError &&
									sopDoc &&
									[
										{
											title: t('v2_sop_doc_section'),
											content: sopDoc.bodyMarkdown,
										},
										{
											title: t('v2_sop_customer_template_section'),
											content: sopDoc.customerUpdateTemplate,
										},
										{
											title: t('v2_sop_vendor_template_section'),
											content: sopDoc.vendorRequestTemplate,
										},
									].map(({ title, content }) => (
										<div className="sop-metadata__doc-section" key={title}>
											<div className="sop-metadata__doc-section-title">
												{title}
											</div>
											{content?.trim() ? (
												<div className="sop-metadata__doc-body">
													<MarkdownRenderer
														markdownContent={content}
														variables={{}}
													/>
												</div>
											) : (
												<div className="sop-metadata__doc-empty">
													{t('v2_sop_doc_empty')}
												</div>
											)}
										</div>
									))}
							</div>
						)}
					</div>
				</div>
			</div>
		</div>
	);
}

export default CreateAlertHeader;
