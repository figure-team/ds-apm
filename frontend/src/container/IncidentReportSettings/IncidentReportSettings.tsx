import './IncidentReportSettings.styles.scss';

import { toast } from '@signozhq/ui';
import { Button, Input, Typography } from 'antd';
import generateReport from 'api/incidentReport/generateReport';
import getTemplate from 'api/incidentReport/getTemplate';
import updateTemplate from 'api/incidentReport/updateTemplate';
import { useEffect, useState } from 'react';

const { TextArea } = Input;

function IncidentReportSettings(): JSX.Element {
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
			.catch(() => toast.error('템플릿을 불러오지 못했습니다.'))
			.finally(() => setLoadingTpl(false));
	}, []);

	const handleSaveTemplate = async (): Promise<void> => {
		setSavingTpl(true);
		try {
			const res = await updateTemplate(template);
			setIsDefault(res.isDefault);
			toast.success('보고서 양식을 저장했습니다.');
		} catch (e) {
			toast.error(
				(e as { response?: { data?: { error?: { message?: string } } } })?.response
					?.data?.error?.message ?? '양식 저장에 실패했습니다.',
			);
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
			toast.success('기본 양식으로 초기화했습니다.');
		} catch {
			toast.error('초기화에 실패했습니다.');
		} finally {
			setSavingTpl(false);
		}
	};

	const handleGenerate = async (): Promise<void> => {
		if (!incidentId.trim()) {
			toast.error('인시던트 ID를 입력하세요.');
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
			toast.error(
				(e as { response?: { data?: { error?: { message?: string } } } })?.response
					?.data?.error?.message ?? '보고서 생성에 실패했습니다.',
			);
		} finally {
			setGenerating(false);
		}
	};

	return (
		<div className="incident-report-settings">
			<Typography.Title level={4}>장애보고서</Typography.Title>
			<Typography.Paragraph type="secondary">
				CF-2 대응 전략과 CF-11 코드 RCA 결과를 집약해 한국 SI 양식의 장애보고서를
				생성합니다. 조직별 양식(Go text/template)을 관리하고, 인시던트별로 보고서를
				뽑을 수 있습니다.
			</Typography.Paragraph>

			<section className="incident-report-settings__block">
				<Typography.Title level={5}>
					보고서 양식 템플릿{' '}
					{isDefault && (
						<Typography.Text type="secondary">(기본 양식 사용 중)</Typography.Text>
					)}
				</Typography.Title>
				<Typography.Paragraph type="secondary">
					{`{{.Title}}, {{.RootCause}}, {{range .Hypotheses}} 등 보고서 데이터를 Go text/template 문법으로 배치합니다. {{ph .Field}}는 값이 비면 "확인 중"으로 표시합니다.`}
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
						양식 저장
					</Button>
					<Button onClick={handleResetTemplate} disabled={loadingTpl || savingTpl}>
						기본 양식으로 초기화
					</Button>
				</div>
			</section>

			<section className="incident-report-settings__block">
				<Typography.Title level={5}>보고서 생성</Typography.Title>
				<div className="incident-report-settings__form">
					<label htmlFor="ir-incident-id">
						<span>인시던트 ID *</span>
						<Input
							id="ir-incident-id"
							value={incidentId}
							onChange={(e): void => setIncidentId(e.target.value)}
							placeholder="INC-PAY-DEMO-2"
						/>
					</label>
					<label htmlFor="ir-service">
						<span>서비스</span>
						<Input
							id="ir-service"
							value={service}
							onChange={(e): void => setService(e.target.value)}
							placeholder="payment-api"
						/>
					</label>
					<label htmlFor="ir-fingerprint">
						<span>알람 fingerprint</span>
						<Input
							id="ir-fingerprint"
							value={alertFingerprint}
							onChange={(e): void => setAlertFingerprint(e.target.value)}
							placeholder="fp-pay-5xx"
						/>
					</label>
					<label htmlFor="ir-severity">
						<span>심각도</span>
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
						보고서 생성
					</Button>
					{markdown && (
						<Button
							onClick={(): void => {
								navigator.clipboard.writeText(markdown).then(
									() => toast.success('복사했습니다.'),
									() => toast.error('복사 실패'),
								);
							}}
						>
							마크다운 복사
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
