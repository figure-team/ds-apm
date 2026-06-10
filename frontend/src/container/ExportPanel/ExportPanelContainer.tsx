import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from 'react-query';
import { Button, Typography } from 'antd';
import createDashboard from 'api/v1/dashboards/create';
import { ENTITY_VERSION_V5 } from 'constants/app';
import { useGetAllDashboard } from 'hooks/dashboard/useGetAllDashboard';
import { useErrorModal } from 'providers/ErrorModalProvider';
import APIError from 'types/api/error';

import { ExportPanelProps } from '.';
import {
	DashboardSelect,
	NewDashboardButton,
	SelectWrapper,
	Title,
	Wrapper,
} from './styles';
import { filterOptions, getSelectOptions } from './utils';

function ExportPanelContainer({
	isLoading,
	onExport,
}: ExportPanelProps): JSX.Element {
	const { t } = useTranslation(['dashboard']);

	const [dashboardId, setDashboardId] = useState<string | null>(null);

	const {
		data,
		isLoading: isAllDashboardsLoading,
		refetch,
	} = useGetAllDashboard();

	const { showErrorModal } = useErrorModal();

	const { mutate: createNewDashboard, isLoading: createDashboardLoading } =
		useMutation(createDashboard, {
			onSuccess: (data) => {
				if (data.data) {
					onExport(data?.data, true);
				}
				refetch();
			},
			onError: (error) => {
				showErrorModal(error as APIError);
			},
		});

	const options = useMemo(() => getSelectOptions(data?.data || []), [data]);

	const handleExportClick = useCallback((): void => {
		const currentSelectedDashboard = data?.data?.find(
			({ id }) => id === dashboardId,
		);

		onExport(currentSelectedDashboard || null, false);
	}, [data, dashboardId, onExport]);

	const handleSelect = useCallback(
		(selectedDashboardId: string): void => {
			setDashboardId(selectedDashboardId);
		},
		[setDashboardId],
	);

	const handleNewDashboard = useCallback(async () => {
		try {
			await createNewDashboard({
				title: t('new_dashboard_title', {
					ns: 'dashboard',
				}),
				uploadedGrafana: false,
				version: ENTITY_VERSION_V5,
			});
		} catch (error) {
			showErrorModal(error as APIError);
		}
	}, [createNewDashboard, t, showErrorModal]);

	const isDashboardLoading = isAllDashboardsLoading || createDashboardLoading;

	const isDisabled =
		isAllDashboardsLoading || !options?.length || !dashboardId || isLoading;

	return (
		<Wrapper direction="vertical">
			<Title>{t('export_panel')}</Title>

			<SelectWrapper direction="horizontal">
				<DashboardSelect
					placeholder={t('select_dashboard')}
					options={options}
					showSearch
					loading={isDashboardLoading}
					disabled={isDashboardLoading}
					value={dashboardId}
					onSelect={handleSelect}
					filterOption={filterOptions}
				/>
				<Button
					type="primary"
					loading={isLoading}
					disabled={isDisabled}
					onClick={handleExportClick}
				>
					{t('export')}
				</Button>
			</SelectWrapper>

			<Typography>
				{t('or_create_dashboard_with_panel')}
				<NewDashboardButton
					disabled={createDashboardLoading}
					loading={createDashboardLoading}
					type="link"
					onClick={handleNewDashboard}
				>
					{t('new_dashboard')}
				</NewDashboardButton>
			</Typography>
		</Wrapper>
	);
}

export default ExportPanelContainer;
