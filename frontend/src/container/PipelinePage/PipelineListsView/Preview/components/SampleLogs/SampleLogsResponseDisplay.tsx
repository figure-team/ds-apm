import { useTranslation } from 'react-i18next';
import { SampleLogsResponse } from '../../hooks/useSampleLogs';
import LogsList from '../LogsList';

function SampleLogsResponseDisplay({
	response,
}: SampleLogsResponseDisplayProps): JSX.Element {
	const { t } = useTranslation(['pipeline']);
	const { isLoading, isError, logs } = response;

	if (isError) {
		return (
			<div className="sample-logs-notice-container">
				{t('pipeline:sample_logs_error')}
			</div>
		);
	}

	if (isLoading) {
		return <div className="sample-logs-notice-container">{t('pipeline:loading')}</div>;
	}

	if (logs.length < 1) {
		return <div className="sample-logs-notice-container">{t('pipeline:no_logs_found')}</div>;
	}

	return <LogsList logs={logs} />;
}

export interface SampleLogsResponseDisplayProps {
	response: SampleLogsResponse;
}

export default SampleLogsResponseDisplay;
