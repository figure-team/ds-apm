import { useTranslation } from 'react-i18next';
import { ActionMode } from 'types/api/pipeline/def';

import { ModeAndConfigWrapper } from './styles';

function ModeAndConfiguration({
	isActionMode,
	version,
}: ModeAndConfigurationType): JSX.Element {
	const { t } = useTranslation(['pipeline']);
	const actionMode = isActionMode === ActionMode.Editing;

	return (
		<ModeAndConfigWrapper>
			{t('pipeline:mode_label')} <span>{actionMode ? 'Editing' : 'Viewing'}</span>
			<div>{t('pipeline:configuration_version_label')} {version}</div>
		</ModeAndConfigWrapper>
	);
}

export interface ModeAndConfigurationType {
	isActionMode: string;
	version: string | number;
}

export default ModeAndConfiguration;
