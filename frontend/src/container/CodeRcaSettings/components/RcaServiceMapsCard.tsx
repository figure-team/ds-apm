import { type Dispatch, type SetStateAction, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from '@signozhq/ui';
import {
	Alert,
	Button,
	Form,
	Input,
	Popconfirm,
	Select,
	Table,
	Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import deleteServiceMap from 'api/codeRca/deleteServiceMap';
import listServiceMaps from 'api/codeRca/listServiceMaps';
import upsertServiceMap from 'api/codeRca/upsertServiceMap';
import { CodebaseRepo, CodebaseServiceMap } from 'api/codeRca/types';

type Props = {
	/** artifactPath 열이 매핑된 저장소를 되짚어야 해서 목록을 받는다. */
	repos: CodebaseRepo[];
	serviceMaps: CodebaseServiceMap[];
	setServiceMaps: Dispatch<SetStateAction<CodebaseServiceMap[]>>;
	isAdmin: boolean;
};

/** 서비스 이름 → 저장소 매핑을 추가·삭제하는 카드. */
function RcaServiceMapsCard({
	repos,
	serviceMaps,
	setServiceMaps,
	isAdmin,
}: Props): JSX.Element {
	const { t } = useTranslation(['codeRca']);
	const [mapForm] = Form.useForm();

	const handleAddMap = useCallback(async (): Promise<void> => {
		try {
			const values = await mapForm.validateFields();
			const payload: CodebaseServiceMap = {
				orgId: '',
				serviceName: values.serviceName,
				repoId: values.repoId,
				subpath: values.subpath ?? '',
			};
			await upsertServiceMap(payload);
			const mapsRes = await listServiceMaps();
			setServiceMaps(mapsRes.data);
			mapForm.resetFields();
			toast.success(t('saved'));
		} catch (err: unknown) {
			if (err && typeof err === 'object' && 'errorFields' in err) {
				return;
			}
			toast.error(t('save_failed'));
		}
	}, [mapForm, setServiceMaps, t]);

	const handleDeleteMap = useCallback(
		async (serviceName: string): Promise<void> => {
			try {
				await deleteServiceMap(serviceName);
				setServiceMaps((prev) => prev.filter((m) => m.serviceName !== serviceName));
				toast.success(t('saved'));
			} catch {
				toast.error(t('save_failed'));
			}
		},
		[setServiceMaps, t],
	);

	const mapColumns: ColumnsType<CodebaseServiceMap> = [
		{ title: t('map_service'), dataIndex: 'serviceName', key: 'serviceName' },
		{ title: t('map_repo'), dataIndex: 'repoId', key: 'repoId' },
		{
			title: t('map_artifact_path'),
			key: 'artifactPath',
			render: (_: unknown, row: CodebaseServiceMap): JSX.Element => {
				const path = repos.find((r) => r.repoId === row.repoId)?.artifactPath;
				return path ? (
					<span>{path}</span>
				) : (
					<Tag color="warning">{t('map_artifact_path_unset')}</Tag>
				);
			},
		},
		{
			title: '',
			key: 'actions',
			render: (_: unknown, row: CodebaseServiceMap): JSX.Element => (
				<Popconfirm
					title={t('map_delete_confirm')}
					onConfirm={(): Promise<void> => handleDeleteMap(row.serviceName)}
					disabled={!isAdmin}
				>
					<Button size="small" danger disabled={!isAdmin}>
						{t('delete')}
					</Button>
				</Popconfirm>
			),
		},
	];

	return (
		<section className="code-rca-settings__card">
			<h3 className="code-rca-settings__card-title">{t('maps_title')}</h3>

			<Alert
				type="info"
				showIcon
				message={
					<span style={{ wordBreak: 'keep-all', overflowWrap: 'break-word' }}>
						{t('maps_export_hint')}
					</span>
				}
				style={{ marginBottom: 12 }}
			/>

			<Form form={mapForm} layout="inline" style={{ marginBottom: 12 }}>
				<Form.Item name="serviceName" rules={[{ required: true }]}>
					<Input placeholder={t('map_service')} disabled={!isAdmin} />
				</Form.Item>
				<Form.Item name="repoId" rules={[{ required: true }]}>
					<Select
						placeholder={t('map_repo')}
						disabled={!isAdmin}
						style={{ width: 200 }}
						showSearch
						options={repos.map((r) => ({ value: r.repoId, label: r.repoId }))}
					/>
				</Form.Item>
				<Form.Item name="subpath">
					<Input placeholder={t('map_subpath')} disabled={!isAdmin} />
				</Form.Item>
				<Form.Item>
					<Button onClick={handleAddMap} disabled={!isAdmin}>
						{t('map_add')}
					</Button>
				</Form.Item>
			</Form>

			<Table
				dataSource={serviceMaps}
				columns={mapColumns}
				rowKey="serviceName"
				size="small"
				pagination={false}
			/>
		</section>
	);
}

export default RcaServiceMapsCard;
