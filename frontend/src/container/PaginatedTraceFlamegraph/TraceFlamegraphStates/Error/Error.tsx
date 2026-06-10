import { Tooltip, Typography } from 'antd';
import { AxiosError } from 'axios';
import { useTranslation } from 'react-i18next';

import noDataUrl from '@/assets/Icons/no-data.svg';

import './Error.styles.scss';

interface IErrorProps {
	error: AxiosError;
}

function Error(props: IErrorProps): JSX.Element {
	const { error } = props;
	const { t } = useTranslation(['trace']);

	return (
		<div className="error-flamegraph">
			<img
				src={noDataUrl}
				alt="error-flamegraph"
				className="error-flamegraph-img"
			/>
			<Tooltip title={error?.message}>
				<Typography.Text className="no-data-text">
					{error?.message || t('something_went_wrong')}
				</Typography.Text>
			</Tooltip>
		</div>
	);
}

export default Error;
