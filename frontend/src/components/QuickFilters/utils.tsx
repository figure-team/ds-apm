import { SIGNAL_DATA_SOURCE_MAP } from 'components/QuickFilters/QuickFiltersSettings/constants';
import { Filter as FilterType } from 'types/api/quickFilters/getCustomFilters';

import { FiltersType, IQuickFiltersConfig, SignalType } from './types';

export type FilterTitleTranslate = (key: string) => string;

// 자주 쓰는 속성 키는 로케일 키로 매핑하고, 그 외에는 title-case 폴백을 유지한다.
const FILTER_TITLE_MAP: Record<string, string> = {
	duration_nano: 'common:qf_duration',
	durationNano: 'common:qf_duration',
	hasError: 'common:qf_has_error',
	has_error: 'common:qf_has_error',
	'service.name': 'common:qf_service_name',
	serviceName: 'common:qf_service_name',
	'deployment.environment': 'common:qf_deployment_environment',
	environment: 'common:qf_environment',
	name: 'common:qf_name',
	'rpc.method': 'common:qf_rpc_method',
	rpcMethod: 'common:qf_rpc_method',
	response_status_code: 'common:qf_response_status_code',
	responseStatusCode: 'common:qf_response_status_code',
	'http.host': 'common:qf_http_host',
	httpHost: 'common:qf_http_host',
	'http.method': 'common:qf_http_method',
	httpMethod: 'common:qf_http_method',
	'http.route': 'common:qf_http_route',
	httpRoute: 'common:qf_http_route',
	'http.url': 'common:qf_http_url',
	httpUrl: 'common:qf_http_url',
	trace_id: 'common:qf_trace_id',
	traceID: 'common:qf_trace_id',
	severity_text: 'common:qf_severity_text',
	'host.name': 'common:qf_host_name',
	'os.type': 'common:qf_os_type',
	'k8s.cluster.name': 'common:qf_k8s_cluster_name',
	'k8s.deployment.name': 'common:qf_k8s_deployment_name',
	'k8s.namespace.name': 'common:qf_k8s_namespace_name',
	'k8s.pod.name': 'common:qf_k8s_pod_name',
	'k8s.node.name': 'common:qf_k8s_node_name',
};

const FILTER_TYPE_MAP: Record<string, FiltersType> = {
	duration_nano: FiltersType.DURATION,
};

const getFilterName = (str: string, translate: FilterTitleTranslate): string => {
	if (FILTER_TITLE_MAP[str]) {
		// 렌더 트리(useFilterConfig)에서 주입된 t()로 번역한다 — i18next 싱글턴을
		// 여기서 직접 부르면 jest 환경(미초기화 싱글턴)에서 렌더가 깨진다.
		return translate(FILTER_TITLE_MAP[str]);
	}
	// replace . and _ with space
	// capitalize the first letter of each word
	return str
		.replace(/\./g, ' ')
		.replace(/_/g, ' ')
		.split(' ')
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
};

const getFilterType = (att: FilterType): FiltersType => {
	if (FILTER_TYPE_MAP[att.key]) {
		return FILTER_TYPE_MAP[att.key];
	}
	return FiltersType.CHECKBOX;
};

export const getFilterConfig = (
	signal?: SignalType,
	customFilters?: FilterType[],
	config?: IQuickFiltersConfig[],
	translate: FilterTitleTranslate = (key): string => key,
): IQuickFiltersConfig[] => {
	if (!customFilters?.length || !signal) {
		return config || [];
	}

	return customFilters.map(
		(att, index) =>
			({
				type: getFilterType(att),
				title: getFilterName(att.key, translate),
				dataSource: SIGNAL_DATA_SOURCE_MAP[signal],
				attributeKey: {
					id: att.key,
					key: att.key,
					dataType: att.dataType,
					type: att.type,
				},
				defaultOpen: index < 2,
			}) as IQuickFiltersConfig,
	);
};
