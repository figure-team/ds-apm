import { Tooltip, Typography } from 'antd';
import { AxiosError } from 'axios';
import { useTranslation } from 'react-i18next';

import './Error.styles.scss';

interface IErrorProps {
	error: AxiosError;
}

function Error(props: IErrorProps): JSX.Element {
	const { error } = props;
	const { t } = useTranslation(['trace']);

	return (
		<div className="error-waterfall">
			<Typography.Text className="text">{t('something_went_wrong')}</Typography.Text>
			<Tooltip title={error?.message}>
				<Typography.Text className="value" ellipsis>
					{error?.message}
				</Typography.Text>
			</Tooltip>
		</div>
	);
}

export default Error;
