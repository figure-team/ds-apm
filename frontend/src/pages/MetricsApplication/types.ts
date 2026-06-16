import { UseMutateAsyncFunction } from 'react-query';
import type { NotificationInstance } from 'antd/es/notification/interface';
import {
	ApDexPayloadAndSettingsProps,
	SetApDexPayloadProps,
} from 'types/api/metrics/getApDex';

export enum MetricsApplicationTab {
	OVER_METRICS = 'OVER_METRICS',
	DB_CALL_METRICS = 'DB_CALL_METRICS',
	EXTERNAL_METRICS = 'EXTERNAL_METRICS',
}

export interface OnSaveApDexSettingsProps {
	thresholdValue: number;
	servicename: string;
	notifications: NotificationInstance;
	refetchGetApDexSetting?: VoidFunction;
	mutateAsync: UseMutateAsyncFunction<
		SetApDexPayloadProps,
		Error,
		ApDexPayloadAndSettingsProps
	>;
	handlePopOverClose: VoidFunction;
}
