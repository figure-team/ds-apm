import { ChangeEvent, Dispatch, SetStateAction, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import Input from 'components/Input';

function DashboardName({ setName, name }: DashboardNameProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	const onChangeHandler = useCallback(
		(e: ChangeEvent<HTMLInputElement>) => {
			setName(e.target.value);
		},
		[setName],
	);

	return (
		<Input
			size="middle"
			placeholder={t('title_placeholder')}
			value={name}
			onChangeHandler={onChangeHandler}
		/>
	);
}

interface DashboardNameProps {
	name: string;
	setName: Dispatch<SetStateAction<string>>;
}

export default DashboardName;
