import { useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from 'react-query';
import * as Sentry from '@sentry/react';
import type { TabsProps } from 'antd';
import { Tabs } from 'antd';
import getPipeline from 'api/pipeline/get';
import Spinner from 'components/Spinner';
import ChangeHistory from 'container/PipelinePage/Layouts/ChangeHistory';
import PipelinePage from 'container/PipelinePage/Layouts/Pipeline';
import { useNotifications } from 'hooks/useNotifications';
import ErrorBoundaryFallback from 'pages/ErrorBoundaryFallback/ErrorBoundaryFallback';
import { SuccessResponse } from 'types/api';
import { Pipeline } from 'types/api/pipeline/def';

import './Pipelines.styles.scss';

const pipelineRefetchInterval = (
	pipelineResponse: SuccessResponse<Pipeline> | undefined,
): number | false => {
	// Refetch pipeline data periodically if deployment of
	// its latest changes is not complete yet.
	const latestVersion = pipelineResponse?.payload?.history?.[0];
	const isLatestDeploymentFinished = ['DEPLOYED', 'FAILED'].includes(
		latestVersion?.deployStatus || '',
	);
	if (latestVersion && !isLatestDeploymentFinished) {
		return 3000;
	}
	return false;
};

function Pipelines(): JSX.Element {
	const { t } = useTranslation(['pipeline', 'common']);
	const { notifications } = useNotifications();
	const {
		isLoading,
		data: pipelineData,
		isError,
		refetch: refetchPipelineLists,
	} = useQuery(['version', 'latest', 'pipeline'], {
		queryFn: () =>
			getPipeline({
				version: 'latest',
			}),
		refetchInterval: pipelineRefetchInterval,
	});

	const tabItems: TabsProps['items'] = useMemo(
		() => [
			{
				key: 'pipelines',
				label: t('tab_pipelines'),
				children: (
					<PipelinePage
						refetchPipelineLists={refetchPipelineLists}
						pipelineData={pipelineData?.payload as Pipeline}
					/>
				),
			},
			{
				key: 'change-history',
				label: t('tab_change_history'),
				children: (
					<ChangeHistory pipelineData={pipelineData?.payload as Pipeline} />
				),
			},
		],
		[pipelineData?.payload, refetchPipelineLists, t],
	);

	useEffect(() => {
		if (pipelineData?.error && isError) {
			notifications.error({
				message: pipelineData?.error || t('common:something_went_wrong'),
			});
		}
	}, [isError, notifications, pipelineData?.error, t]);

	if (isLoading) {
		return <Spinner height="75vh" tip="Loading Pipelines..." />;
	}

	return (
		<Sentry.ErrorBoundary fallback={<ErrorBoundaryFallback />}>
			<Tabs
				className="pipeline-tabs"
				defaultActiveKey="pipelines"
				items={tabItems}
			/>
		</Sentry.ErrorBoundary>
	);
}

export default Pipelines;
