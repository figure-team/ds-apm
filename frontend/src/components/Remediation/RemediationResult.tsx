import { useTranslation } from 'react-i18next';
import { RemediationExecution } from 'api/remediation';

interface Props {
	rem: RemediationExecution;
}

function RemediationResult({ rem }: Props): JSX.Element {
	const { t } = useTranslation('alerts');
	return (
		<>
			{typeof rem.exitCode === 'number' && (
				<div className="remediation-card__exit">
					{t('remediation_exit_code')}: {rem.exitCode}
				</div>
			)}
			<pre className="remediation-card__output">
				{rem.outputSnippet || t('remediation_no_output')}
			</pre>
			{rem.verifyResult && (
				<div className="remediation-card__verify">{rem.verifyResult}</div>
			)}
			<details className="remediation-card__script-toggle">
				<summary>{t('remediation_show_script')}</summary>
				<pre className="remediation-card__script">{rem.scriptSnapshot}</pre>
			</details>
		</>
	);
}

export default RemediationResult;
