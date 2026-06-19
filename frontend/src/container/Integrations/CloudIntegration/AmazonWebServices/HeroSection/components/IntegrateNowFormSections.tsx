import { Dispatch, SetStateAction } from 'react';
import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Form, Select } from 'antd';
import { ChevronDown } from 'lucide-react';
import { Region } from 'utils/regions';
import { popupContainer } from 'utils/selectPopupContainer';

import { RegionSelector } from './RegionSelector';

// Form section components
function RegionDeploymentSection({
	regions,
	handleRegionChange,
	isFormDisabled,
}: {
	regions: Region[];
	handleRegionChange: (value: string) => void;
	isFormDisabled: boolean;
}): JSX.Element {
	const { t } = useTranslation('integrations');
	return (
		<div className="cloud-account-setup-form__form-group">
			<div className="cloud-account-setup-form__title">
				{t('setup.deploy_stack_title')}
			</div>
			<div className="cloud-account-setup-form__description">
				{t('setup.deploy_stack_description')}
			</div>
			<Form.Item
				name="region"
				rules={[{ required: true, message: 'Please select a region' }]}
				className="cloud-account-setup-form__form-item"
			>
				<Select
					placeholder={t('setup.region_placeholder')}
					suffixIcon={<ChevronDown size={16} color={Color.BG_VANILLA_400} />}
					className="cloud-account-setup-form__select integrations-select"
					onChange={handleRegionChange}
					disabled={isFormDisabled}
					getPopupContainer={popupContainer}
				>
					{regions.flatMap((region) =>
						region.subRegions.map((subRegion) => (
							<Select.Option key={subRegion.id} value={subRegion.id}>
								{subRegion.displayName}
							</Select.Option>
						)),
					)}
				</Select>
			</Form.Item>
		</div>
	);
}

function MonitoringRegionsSection({
	selectedRegions,
	setSelectedRegions,
	setIncludeAllRegions,
}: {
	selectedRegions: string[];
	setSelectedRegions: Dispatch<SetStateAction<string[]>>;
	setIncludeAllRegions: Dispatch<SetStateAction<boolean>>;
}): JSX.Element {
	const { t } = useTranslation('integrations');
	return (
		<div className="cloud-account-setup-form__form-group">
			<div className="cloud-account-setup-form__title">
				{t('regions.which_regions')}
			</div>
			<div className="cloud-account-setup-form__description">
				{t('regions.choose_regions_long')}
			</div>

			<RegionSelector
				selectedRegions={selectedRegions}
				setSelectedRegions={setSelectedRegions}
				setIncludeAllRegions={setIncludeAllRegions}
			/>
		</div>
	);
}

function ComplianceNote(): JSX.Element {
	const { t } = useTranslation('integrations');
	return (
		<div className="cloud-account-setup-form__form-group">
			<div className="cloud-account-setup-form__note">
				{t('setup.compliance_note')}
			</div>
		</div>
	);
}

export { ComplianceNote, MonitoringRegionsSection, RegionDeploymentSection };
