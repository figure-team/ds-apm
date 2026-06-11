import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Color, Style } from '@signozhq/design-tokens';
import {
	ChevronDown,
	ChevronRight,
	CircleHelp,
	Plus,
	Trash2,
	TriangleAlert,
} from '@signozhq/icons';
import { Button, Checkbox, Input } from '@signozhq/ui';
import { Collapse, Form, Select, Tooltip } from 'antd';
import { useCollapseSectionErrors } from 'hooks/useCollapseSectionErrors';

import './RoleMappingSection.styles.scss';

const ROLE_OPTIONS = [
	{ value: 'VIEWER', label: 'VIEWER' },
	{ value: 'EDITOR', label: 'EDITOR' },
	{ value: 'ADMIN', label: 'ADMIN' },
];

interface RoleMappingSectionProps {
	fieldNamePrefix: string[];
	isExpanded?: boolean;
	onExpandChange?: (expanded: boolean) => void;
}

function RoleMappingSection({
	fieldNamePrefix,
	isExpanded,
	onExpandChange,
}: RoleMappingSectionProps): JSX.Element {
	const form = Form.useFormInstance();
	const { t } = useTranslation('organizationsettings');
	const useRoleAttribute = Form.useWatch(
		[...fieldNamePrefix, 'useRoleAttribute'],
		form,
	);

	// Support both controlled and uncontrolled modes
	const [internalExpanded, setInternalExpanded] = useState(false);
	const isControlled = isExpanded !== undefined;
	const expanded = isControlled ? isExpanded : internalExpanded;

	const handleCollapseChange = useCallback(
		(keys: string | string[]): void => {
			const newExpanded = Array.isArray(keys) ? keys.length > 0 : !!keys;
			if (isControlled && onExpandChange) {
				onExpandChange(newExpanded);
			} else {
				setInternalExpanded(newExpanded);
			}
		},
		[isControlled, onExpandChange],
	);

	const collapseActiveKey = expanded ? ['role-mapping'] : [];
	const { hasErrors, errorMessages } = useCollapseSectionErrors(fieldNamePrefix);

	return (
		<div className="role-mapping-section">
			<Collapse
				bordered={false}
				activeKey={collapseActiveKey}
				onChange={handleCollapseChange}
				className="role-mapping-section__collapse"
				expandIcon={(): null => null}
			>
				<Collapse.Panel
					key="role-mapping"
					header={
						<div
							className="role-mapping-section__collapse-header"
							role="button"
							aria-expanded={expanded}
							aria-controls="role-mapping-content"
						>
							{!expanded ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
							<div className="role-mapping-section__collapse-header-text">
								<h4 className="role-mapping-section__section-title">
									{t('role_mapping_title')}
								</h4>
								<p className="role-mapping-section__section-description">
									{t('role_mapping_description')}
								</p>
							</div>
							{!expanded && hasErrors && (
								<Tooltip
									title={
										<>
											{errorMessages.map((msg) => (
												<div key={msg}>{msg}</div>
											))}
										</>
									}
								>
									<TriangleAlert size={16} color={Color.BG_CHERRY_500} />
								</Tooltip>
							)}
						</div>
					}
				>
					<div id="role-mapping-content" className="role-mapping-section__content">
						<div className="role-mapping-section__field-group">
							<label className="role-mapping-section__label" htmlFor="default-role">
								{t('role_mapping_default_role')}
								<Tooltip title={t('role_mapping_tooltip_default_role')}>
									<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
								</Tooltip>
							</label>
							<Form.Item
								name={[...fieldNamePrefix, 'defaultRole']}
								className="role-mapping-section__form-item"
								initialValue="VIEWER"
							>
								<Select
									id="default-role"
									options={ROLE_OPTIONS}
									className="role-mapping-section__select"
								/>
							</Form.Item>
						</div>

						<div className="role-mapping-section__checkbox-row">
							<Form.Item
								name={[...fieldNamePrefix, 'useRoleAttribute']}
								valuePropName="value"
								noStyle
							>
								<Checkbox
									id="use-role-attribute"
									onChange={(checked: boolean): void => {
										form.setFieldValue([...fieldNamePrefix, 'useRoleAttribute'], checked);
									}}
								>
									{t('role_mapping_use_role_attribute')}
								</Checkbox>
							</Form.Item>
							<Tooltip title={t('role_mapping_tooltip_use_role_attribute')}>
								<CircleHelp size={14} color={Style.L3_FOREGROUND} cursor="help" />
							</Tooltip>
						</div>

						{!useRoleAttribute && (
							<div className="role-mapping-section__group-mappings">
								<div className="role-mapping-section__group-header">
									<span className="role-mapping-section__group-title">
										{t('role_mapping_group_title')}
									</span>
									<p className="role-mapping-section__group-description">
										{t('role_mapping_group_description')}
									</p>
								</div>

								<Form.List name={[...fieldNamePrefix, 'groupMappingsList']}>
									{(fields, { add, remove }): JSX.Element => (
										<div className="role-mapping-section__items">
											{fields.map((field) => (
												<div key={field.key} className="role-mapping-section__row">
													<Form.Item
														name={[field.name, 'groupName']}
														className="role-mapping-section__field role-mapping-section__field--group"
														rules={[
															{
																required: true,
																message: t('role_mapping_group_name_required'),
															},
														]}
													>
														<Input placeholder={t('role_mapping_group_name_placeholder')} />
													</Form.Item>

													<Form.Item
														name={[field.name, 'role']}
														className="role-mapping-section__field role-mapping-section__field--role"
														rules={[
															{ required: true, message: t('role_mapping_role_required') },
														]}
														initialValue="VIEWER"
													>
														<Select
															options={ROLE_OPTIONS}
															className="role-mapping-section__select"
														/>
													</Form.Item>

													<Button
														variant="ghost"
														color="secondary"
														className="role-mapping-section__remove-btn"
														onClick={(): void => remove(field.name)}
														aria-label={t('role_mapping_remove_aria')}
													>
														<Trash2 size={12} />
													</Button>
												</div>
											))}

											<Button
												variant="outlined"
												color="secondary"
												onClick={(): void => add({ groupName: '', role: 'VIEWER' })}
												prefix={<Plus size={14} />}
											>
												{t('role_mapping_add')}
											</Button>
										</div>
									)}
								</Form.List>
							</div>
						)}
					</div>
				</Collapse.Panel>
			</Collapse>
		</div>
	);
}

export default RoleMappingSection;
