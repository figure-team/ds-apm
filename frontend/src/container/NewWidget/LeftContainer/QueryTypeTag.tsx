import { useTranslation } from 'react-i18next';
import { EQueryType } from 'types/common/dashboard';

function QueryTypeTag({ queryType }: IQueryTypeTagProps): JSX.Element {
	const { t } = useTranslation('common');
	switch (queryType) {
		case EQueryType.QUERY_BUILDER:
			return <span>{t('query_builder.type_query_builder')}</span>;

		case EQueryType.CLICKHOUSE:
			return <span>{t('query_builder.type_clickhouse_query')}</span>;
		case EQueryType.PROM:
			return <span>PromQL</span>;
		default:
			return <span />;
	}
}

interface IQueryTypeTagProps {
	queryType?: EQueryType;
}

QueryTypeTag.defaultProps = {
	queryType: EQueryType.QUERY_BUILDER,
};

export default QueryTypeTag;
