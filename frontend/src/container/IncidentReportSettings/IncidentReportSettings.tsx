import './IncidentReportSettings.styles.scss';

import { toast } from '@signozhq/ui';
import { Button, Input, Typography } from 'antd';
import generateReport from 'api/incidentReport/generateReport';
import getTemplate from 'api/incidentReport/getTemplate';
import updateTemplate from 'api/incidentReport/updateTemplate';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getApiErrorMessage } from 'utils/errorUtils';

const { TextArea } = Input;

function IncidentReportSettings(): JSX.Element {
	const { t } = useTranslation(['incident_report']);
	const [template, setTemplate] = useState('');
	const [isDefault, setIsDefault] = useState(true);
	const [loadingTpl, setLoadingTpl] = useState(true);
	const [savingTpl, setSavingTpl] = useState(false);

	const [incidentId, setIncidentId] = useState('');
	const [alertFingerprint, setAlertFingerprint] = useState('');
	const [service, setService] = useState('');
	const [severity, setSeverity] = useState('');
	const [generating, setGenerating] = useState(false);
	const [markdown, setMarkdown] = useState('');

	useEffect(() => {
		getTemplate()
			.then((res) => {
				setTemplate(res.template);
				setIsDefault(res.isDefault);
			})
			.catch(() => toast.error(t('toast_template_load_failed')))
			.finally(() => setLoadingTpl(false));
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	const handleSaveTemplate = async (): Promise<void> => {
		setSavingTpl(true);
		try {
			const res = await updateTemplate(template);
			setIsDefault(res.isDefault);
			toast.success(t('toast_template_saved'));
		} catch (e) {
			toast.error(getApiErrorMessage(e, t('toast_template_save_failed')));
		} finally {
			setSavingTpl(false);
		}
	};

	const handleResetTemplate = async (): Promise<void> => {
		setSavingTpl(true);
		try {
			await updateTemplate('');
			const res = await getTemplate();
			setTemplate(res.template);
			setIsDefault(res.isDefault);
			toast.success(t('toast_template_reset_done'));
		} catch {
			toast.error(t('toast_template_reset_failed'));
		} finally {
			setSavingTpl(false);
		}
	};

	const handleGenerate = async (): Promise<void> => {
		if (!incidentId.trim()) {
			toast.error(t('toast_incident_id_required'));
			return;
		}
		setGenerating(true);
		try {
			const res = await generateReport({
				incidentId: incidentId.trim(),
				alertFingerprint: alertFingerprint.trim() || undefined,
				service: service.trim() || undefined,
				severity: severity.trim() || undefined,
			});
			setMarkdown(res.markdown);
		} catch (e) {
			toast.error(getApiErrorMessage(e, t('toast_generate_failed')));
		} finally {
			setGenerating(false);
		}
	};

	return (
		<div className="incident-report-settings settings-shell settings-shell--narrow">
			<header className="incident-report-settings__header">
				<h1 className="incident-report-settings__header-title">
					{t('header_title')}
				</h1>
				<p className="incident-report-settings__header-subtitle">
					{t('header_subtitle')}
				</p>
			</header>

			<section className="incident-report-settings__block">
				<Typography.Title level={5}>
					{t('section_template_title')}{' '}
					{isDefault && (
						<Typography.Text type="secondary">
							{t('label_default_in_use')}
						</Typography.Text>
					)}
				</Typography.Title>
				<Typography.Paragraph type="secondary">
					{t('template_help')}
				</Typography.Paragraph>
				<TextArea
					value={template}
					onChange={(e): void => setTemplate(e.target.value)}
					rows={14}
					disabled={loadingTpl}
					spellCheck={false}
				/>
				<div className="incident-report-settings__actions">
					<Button
						type="primary"
						loading={savingTpl}
						onClick={handleSaveTemplate}
						disabled={loadingTpl}
					>
						{t('btn_save_template')}
					</Button>
					<Button onClick={handleResetTemplate} disabled={loadingTpl || savingTpl}>
						{t('btn_reset_template')}
					</Button>
				</div>
			</section>

			<section className="incident-report-settings__block">
				<Typography.Title level={5}>
					{t('section_generate_title')}
				</Typography.Title>
				<div className="incident-report-settings__form">
					<label htmlFor="ir-incident-id">
						<span>{t('field_incident_id')} *</span>
						<Input
							id="ir-incident-id"
							value={incidentId}
							onChange={(e): void => setIncidentId(e.target.value)}
							placeholder="INC-PAY-DEMO-2"
						/>
					</label>
					<label htmlFor="ir-service">
						<span>{t('field_service')}</span>
						<Input
							id="ir-service"
							value={service}
							onChange={(e): void => setService(e.target.value)}
							placeholder="payment-api"
						/>
					</label>
					<label htmlFor="ir-fingerprint">
						<span>{t('field_fingerprint')}</span>
						<Input
							id="ir-fingerprint"
							value={alertFingerprint}
							onChange={(e): void => setAlertFingerprint(e.target.value)}
							placeholder="fp-pay-5xx"
						/>
					</label>
					<label htmlFor="ir-severity">
						<span>{t('field_severity')}</span>
						<Input
							id="ir-severity"
							value={severity}
							onChange={(e): void => setSeverity(e.target.value)}
							placeholder="critical"
						/>
					</label>
				</div>
				<div className="incident-report-settings__actions">
					<Button type="primary" loading={generating} onClick={handleGenerate}>
						{t('btn_generate')}
					</Button>
					{markdown && (
						<Button
							onClick={(): void => {
								navigator.clipboard.writeText(markdown).then(
									() => toast.success(t('toast_copy_done')),
									() => toast.error(t('toast_copy_failed')),
								);
							}}
						>
							{t('btn_copy_markdown')}
						</Button>
					)}
				</div>
				{markdown && (
					<pre className="incident-report-settings__result">{markdown}</pre>
				)}
			</section>
		</div>
	);
}

export default IncidentReportSettings;
