import {
	CheckCircleFilled,
	CloseCircleFilled,
	ExclamationCircleFilled,
	LoadingOutlined,
	MinusCircleFilled,
} from '@ant-design/icons';
import { Spin } from 'antd';
import { TFunction } from 'i18next';

export function getDeploymentStage(value: string, t: TFunction): string {
	switch (value) {
		case 'in_progress':
			return t('deploy_stage_in_progress').toString();
		case 'deployed':
			return t('deploy_stage_deployed').toString();
		case 'dirty':
			return t('deploy_stage_dirty').toString();
		case 'failed':
			return t('deploy_stage_failed').toString();
		case 'unknown':
			return t('deploy_stage_unknown').toString();
		default:
			return '';
	}
}

export function getDeploymentStageIcon(value: string): JSX.Element {
	switch (value) {
		case 'in_progress':
			return (
				<Spin indicator={<LoadingOutlined style={{ fontSize: 15 }} spin />} />
			);
		case 'deployed':
			return <CheckCircleFilled />;
		case 'dirty':
			return <ExclamationCircleFilled />;
		case 'failed':
			return <CloseCircleFilled />;
		case 'unknown':
			return <MinusCircleFilled />;
		default:
			return <span />;
	}
}
