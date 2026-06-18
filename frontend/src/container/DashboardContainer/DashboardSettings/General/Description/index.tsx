import { useTranslation } from 'react-i18next';
import { ChangeEvent, Dispatch, SetStateAction, useCallback } from 'react';
import { Input } from 'antd';

import { Container } from './styles';

const { TextArea } = Input;

function Description({
	description,
	setDescription,
}: DescriptionProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	const onChangeHandler = useCallback(
		(e: ChangeEvent<HTMLTextAreaElement>) => {
			setDescription(e.target.value);
		},
		[setDescription],
	);

	return (
		<Container>
			<TextArea
				placeholder={t('description_of_dashboard')}
				onChange={onChangeHandler}
				value={description}
			/>
		</Container>
	);
}

interface DescriptionProps {
	description: string;
	setDescription: Dispatch<SetStateAction<string>>;
}

export default Description;
