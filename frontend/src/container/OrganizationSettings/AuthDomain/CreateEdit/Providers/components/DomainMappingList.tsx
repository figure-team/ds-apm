import { Plus, Trash2 } from '@signozhq/icons';
import { Button, Input } from '@signozhq/ui';
import { Form } from 'antd';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

import './DomainMappingList.styles.scss';

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const createValidateEmail = (t: TFunction) => (
	_: unknown,
	value: string,
): Promise<void> => {
	if (!value) {
		return Promise.reject(new Error(t('domain_mapping_admin_email_required')));
	}
	if (!EMAIL_REGEX.test(value)) {
		return Promise.reject(new Error(t('domain_mapping_invalid_email')));
	}
	return Promise.resolve();
};

interface DomainMappingListProps {
	fieldNamePrefix: string[];
}

function DomainMappingList({
	fieldNamePrefix,
}: DomainMappingListProps): JSX.Element {
	const { t } = useTranslation('organizationsettings');
	const validateEmail = createValidateEmail(t);
	return (
		<div className="domain-mapping-list">
			<div className="domain-mapping-list__header">
				<span className="domain-mapping-list__title">
					{t('domain_mapping_title')}
				</span>
				<p className="domain-mapping-list__description">
					{t('domain_mapping_description')}
				</p>
			</div>

			<Form.List name={fieldNamePrefix}>
				{(fields, { add, remove }): JSX.Element => (
					<div className="domain-mapping-list__items">
						{fields.map((field) => (
							<div key={field.key} className="domain-mapping-list__row">
								<Form.Item
									name={[field.name, 'domain']}
									className="domain-mapping-list__field"
									rules={[
										{ required: true, message: t('domain_mapping_domain_required') },
									]}
								>
									<Input placeholder={t('domain_mapping_domain_placeholder')} />
								</Form.Item>

								<Form.Item
									name={[field.name, 'adminEmail']}
									className="domain-mapping-list__field"
									rules={[{ validator: validateEmail }]}
								>
									<Input placeholder={t('domain_mapping_admin_email_placeholder')} />
								</Form.Item>

								<Button
									variant="ghost"
									color="secondary"
									className="domain-mapping-list__remove-btn"
									onClick={(): void => remove(field.name)}
									aria-label={t('domain_mapping_remove_aria')}
								>
									<Trash2 size={12} />
								</Button>
							</div>
						))}

						<Button
							variant="dashed"
							onClick={(): void => add({ domain: '', adminEmail: '' })}
							prefix={<Plus size={14} />}
							className="domain-mapping-list__add-btn"
						>
							{t('domain_mapping_add')}
						</Button>
					</div>
				)}
			</Form.List>
		</div>
	);
}

export default DomainMappingList;
