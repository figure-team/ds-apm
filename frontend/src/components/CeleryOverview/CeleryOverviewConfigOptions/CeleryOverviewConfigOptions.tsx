import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useHistory, useLocation } from 'react-router-dom';
import { Row, Select, Spin } from 'antd';
import {
	getValuesFromQueryParams,
	setQueryParamsFromOptions,
} from 'components/CeleryTask/CeleryUtils';
import { useCeleryFilterOptions } from 'components/CeleryTask/useCeleryFilterOptions';
import { SelectMaxTagPlaceholder } from 'components/MessagingQueues/MQCommon/MQCommon';
import { QueryParams } from 'constants/query';
import useUrlQuery from 'hooks/useUrlQuery';

import './CeleryOverviewConfigOptions.styles.scss';

export interface SelectOptionConfig {
	placeholder: string;
	queryParam: QueryParams;
	filterType: string | string[];
	shouldSetQueryParams?: boolean;
	onChange?: (value: string | string[]) => void;
	values?: string | string[];
	isMultiple?: boolean;
}

export function FilterSelect({
	placeholder,
	queryParam,
	filterType,
	values,
	shouldSetQueryParams,
	onChange,
	isMultiple,
}: SelectOptionConfig): JSX.Element {
	const { t } = useTranslation('messagingQueues');
	const { handleSearch, isFetching, options } =
		useCeleryFilterOptions(filterType);

	const urlQuery = useUrlQuery();
	const history = useHistory();
	const location = useLocation();

	// Add state to track the current search input
	const [searchValue, setSearchValue] = useState<string>('');

	// Use externally provided `values` if `shouldSetQueryParams` is false, otherwise get from URL params.
	const selectValue =
		!shouldSetQueryParams && !!values?.length
			? values
			: getValuesFromQueryParams(queryParam, urlQuery) || [];

	// Memoize options to include the typed value if not present
	const mergedOptions = useMemo(() => {
		if (
			!!searchValue.trim().length &&
			!options.some((opt) => opt.value === searchValue)
		) {
			return [{ value: searchValue, label: searchValue }, ...options];
		}
		return options;
	}, [options, searchValue]);

	const handleSelectChange = useCallback(
		(value: string | string[]): void => {
			handleSearch('');
			setSearchValue(''); // Clear search value after selection
			if (shouldSetQueryParams) {
				setQueryParamsFromOptions(
					value as string[],
					urlQuery,
					history,
					location,
					queryParam,
				);
			}
			onChange?.(value);
		},
		[
			handleSearch,
			shouldSetQueryParams,
			urlQuery,
			history,
			location,
			queryParam,
			onChange,
		],
	);

	// Update searchValue on user input
	const handleSearchInput = (input: string): void => {
		setSearchValue(input);
		handleSearch(input);
	};

	return (
		<Select
			key={filterType.toString()}
			placeholder={placeholder}
			showSearch
			{...(isMultiple ? { mode: 'multiple' } : {})}
			options={mergedOptions}
			loading={isFetching}
			className="config-select-option"
			onSearch={handleSearchInput}
			maxTagCount={4}
			allowClear
			maxTagPlaceholder={SelectMaxTagPlaceholder}
			value={selectValue}
			notFoundContent={
				isFetching ? (
					<span>
						<Spin size="small" /> {t('loading')}
					</span>
				) : (
					<span>{t('no_filter_found', { filter: placeholder })}</span>
				)
			}
			onChange={handleSelectChange}
		/>
	);
}

FilterSelect.defaultProps = {
	shouldSetQueryParams: true,
	onChange: (): void => {},
	values: [],
	isMultiple: true,
};

function CeleryOverviewConfigOptions(): JSX.Element {
	const { t } = useTranslation('messagingQueues');
	const selectConfigs: SelectOptionConfig[] = [
		{
			placeholder: t('filter_service_name'),
			queryParam: QueryParams.service,
			filterType: 'serviceName',
		},
		{
			placeholder: t('filter_span_name'),
			queryParam: QueryParams.spanName,
			filterType: 'name',
		},
		{
			placeholder: t('filter_msg_system'),
			queryParam: QueryParams.msgSystem,
			filterType: 'messaging.system',
		},
		{
			placeholder: t('filter_destination'),
			queryParam: QueryParams.destination,
			filterType: ['messaging.destination.name', 'messaging.destination'],
		},
		{
			placeholder: t('filter_kind'),
			queryParam: QueryParams.kindString,
			filterType: 'kind_string',
		},
	];

	return (
		<div className="celery-overview-filters">
			<Row className="celery-filters">
				{selectConfigs.map((config) => (
					<FilterSelect
						key={config.filterType.toString()}
						placeholder={config.placeholder}
						queryParam={config.queryParam}
						filterType={config.filterType}
					/>
				))}
			</Row>
		</div>
	);
}

export default CeleryOverviewConfigOptions;
