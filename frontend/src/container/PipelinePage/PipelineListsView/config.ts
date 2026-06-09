import {
	TableColumnGroupType as ColumnGroupType,
	TableColumnType as ColumnType,
} from 'antd/';
import { TFunction } from 'i18next';
import {
	HistoryData,
	PipelineData,
	ProcessorData,
} from 'types/api/pipeline/def';

import DeploymentStage from '../Layouts/ChangeHistory/DeploymentStage';
import DeploymentTime from '../Layouts/ChangeHistory/DeploymentTime';
import DescriptionTextArea from './AddNewPipeline/FormFields/DescriptionTextArea';
import FilterInput from './AddNewPipeline/FormFields/FilterInput';
import NameInput from './AddNewPipeline/FormFields/NameInput';

export const pipelineFields = [
	{
		id: 1,
		fieldName: 'Name',
		placeholder: 'pipeline_name_placeholder',
		name: 'name',
		component: NameInput,
	},
	{
		id: 2,
		fieldName: 'Description',
		placeholder: 'pipeline_description_placeholder',
		name: 'description',
		component: DescriptionTextArea,
	},
	{
		id: 3,
		fieldName: 'Filter',
		placeholder: 'pipeline_filter_placeholder',
		name: 'filter',
		component: FilterInput,
	},
];

export const tagInputStyle: React.CSSProperties = {
	width: 78,
	verticalAlign: 'top',
	flex: 1,
};

export const pipelineColumns: Array<
	ColumnType<PipelineData> | ColumnGroupType<PipelineData>
> = [
	{
		key: 'orderId',
		title: '',
		dataIndex: 'orderId',
	},
	{
		key: 'name',
		title: 'Pipeline Name',
		dataIndex: 'name',
	},
	{
		key: 'filter',
		title: 'Filters',
		dataIndex: 'filter',
	},

	{
		key: 'createdAt',
		title: 'Last Edited',
		dataIndex: 'createdAt',
	},
	{
		key: 'createdBy',
		title: 'Edited By',
		dataIndex: 'createdBy',
	},
];

export const processorColumns: Array<
	ColumnType<ProcessorData> | ColumnGroupType<ProcessorData>
> = [
	{
		key: 'id',
		title: '',
		dataIndex: 'orderId',
		width: 150,
	},
	{
		key: 'name',
		title: '',
		dataIndex: 'name',
	},
];

export const getChangeHistoryColumns = (
	t: TFunction,
): Array<ColumnType<HistoryData> | ColumnGroupType<HistoryData>> => [
	{
		key: 'version',
		title: t('column_version').toString(),
		dataIndex: 'version',
	},
	{
		title: t('column_deployment_stage').toString(),
		key: 'deployStatus',
		dataIndex: 'deployStatus',
		render: (deployStatus: string): JSX.Element =>
			DeploymentStage(deployStatus, t),
	},
	{
		key: 'deployResult',
		title: t('column_last_deploy_message').toString(),
		dataIndex: 'deployResult',
		ellipsis: true,
	},
	{
		key: 'createdAt',
		title: t('column_last_deployed_time').toString(),
		dataIndex: 'createdAt',
		render: DeploymentTime,
	},
	{
		key: 'createdByName',
		title: t('column_edited_by').toString(),
		dataIndex: 'createdByName',
	},
];

export const formValidationRules = [
	{
		required: true,
	},
];

export const iconStyle = { fontSize: '1rem' };
export const smallIconStyle = { fontSize: '0.75rem' };
export const holdIconStyle = { ...iconStyle, cursor: 'move' };
