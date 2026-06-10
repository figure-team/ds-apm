import { IconDataSpan } from 'container/PipelinePage/styles';
import { TFunction } from 'i18next';

import { getDeploymentStage, getDeploymentStageIcon } from './utils';

function DeploymentStage(deployStatus: string, t: TFunction): JSX.Element {
	return (
		<>
			{getDeploymentStageIcon(deployStatus)}
			<IconDataSpan>{getDeploymentStage(deployStatus, t)}</IconDataSpan>
		</>
	);
}

export default DeploymentStage;
