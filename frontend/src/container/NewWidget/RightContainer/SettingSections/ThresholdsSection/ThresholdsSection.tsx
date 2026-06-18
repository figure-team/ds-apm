import { Dispatch, SetStateAction } from 'react';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { Antenna } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ColumnUnit } from 'types/api/dashboard/getAll';

import SettingsSection from '../../components/SettingsSection/SettingsSection';
import ThresholdSelector from '../../Threshold/ThresholdSelector';
import { ThresholdProps } from '../../Threshold/types';

import './ThresholdsSection.styles.scss';

interface ThresholdsSectionProps {
	thresholds: ThresholdProps[];
	setThresholds: Dispatch<SetStateAction<ThresholdProps[]>>;
	yAxisUnit: string;
	selectedGraph: PANEL_TYPES;
	columnUnits: ColumnUnit;
}

export default function ThresholdsSection({
	thresholds,
	setThresholds,
	yAxisUnit,
	selectedGraph,
	columnUnits,
}: ThresholdsSectionProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<SettingsSection
			title={t('section_thresholds')}
			icon={<Antenna size={14} />}
			defaultOpen={!!thresholds.length}
		>
			<ThresholdSelector
				thresholds={thresholds}
				setThresholds={setThresholds}
				yAxisUnit={yAxisUnit}
				selectedGraph={selectedGraph}
				columnUnits={columnUnits}
			/>
		</SettingsSection>
	);
}
