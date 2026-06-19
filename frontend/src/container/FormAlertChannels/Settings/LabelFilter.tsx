import { Dispatch, SetStateAction } from 'react';
import { Form, Input, Select } from 'antd';
import { LabelFilterStatement } from 'container/CreateAlertChannels/config';
import { useTranslation } from 'react-i18next';

const { Option } = Select;

// LabelFilterForm supports filters or matchers on alert notifications
// presently un-used but will be introduced to the channel creation at some
// point
function LabelFilterForm({ setFilter }: LabelFilterProps): JSX.Element {
	const { t } = useTranslation('channels');
	return (
		<Form.Item name="label_filter" label={t('label_filter_notify_when')}>
			<Input.Group compact>
				<Select
					defaultValue="Severity"
					style={{ width: '15%' }}
					onChange={(event): void => {
						setFilter((value) => {
							const first: LabelFilterStatement = value[0] as LabelFilterStatement;
							first.name = event;
							return [first];
						});
					}}
				>
					<Option value="severity">Severity</Option>
					<Option value="service">Service</Option>
				</Select>
				<Select
					defaultValue="="
					onChange={(event): void => {
						setFilter((value) => {
							const first: LabelFilterStatement = value[0] as LabelFilterStatement;
							first.comparator = event;
							return [first];
						});
					}}
				>
					<Option value="=">=</Option>
					<Option value="!=">!=</Option>
				</Select>
				<Input
					style={{ width: '20%' }}
					placeholder={t('label_filter_placeholder')}
					onChange={(event): void => {
						setFilter((value) => {
							const first: LabelFilterStatement = value[0] as LabelFilterStatement;
							first.value = event.target.value;
							return [first];
						});
					}}
				/>
			</Input.Group>
		</Form.Item>
	);
}

export interface LabelFilterProps {
	setFilter: Dispatch<SetStateAction<Partial<Array<LabelFilterStatement>>>>;
}

export default LabelFilterForm;
