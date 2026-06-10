import { Typography } from 'antd';
import { useTranslation } from 'react-i18next';

import noDataUrl from '@/assets/Icons/no-data.svg';

import './NoData.styles.scss';

interface INoDataProps {
	name: string;
}

function NoData(props: INoDataProps): JSX.Element {
	const { name } = props;
	const { t } = useTranslation(['trace']);

	return (
		<div className="no-data">
			<img src={noDataUrl} alt="no-data" className="no-data-img" />
			<Typography.Text className="no-data-text">
				{t('no_data_found_for_span', { name })}
			</Typography.Text>
		</div>
	);
}

export default NoData;
