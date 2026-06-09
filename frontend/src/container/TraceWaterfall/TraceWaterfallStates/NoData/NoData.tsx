import { Typography } from 'antd';
import { useTranslation } from 'react-i18next';

interface INoDataProps {
	id: string;
}

function NoData(props: INoDataProps): JSX.Element {
	const { id } = props;
	const { t } = useTranslation(['trace']);
	return <Typography.Text>{t('no_trace_found_with_id', { id })}</Typography.Text>;
}

export default NoData;
