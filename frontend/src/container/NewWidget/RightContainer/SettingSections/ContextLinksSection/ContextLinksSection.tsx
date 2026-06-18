import { Dispatch, SetStateAction } from 'react';
import { Link as LinkIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ContextLinksData, Widgets } from 'types/api/dashboard/getAll';

import SettingsSection from '../../components/SettingsSection/SettingsSection';
import ContextLinks from '../../ContextLinks';

import './ContextLinksSection.styles.scss';

interface ContextLinksSectionProps {
	contextLinks: ContextLinksData;
	setContextLinks: Dispatch<SetStateAction<ContextLinksData>>;
	selectedWidget?: Widgets;
}

export default function ContextLinksSection({
	contextLinks,
	setContextLinks,
	selectedWidget,
}: ContextLinksSectionProps): JSX.Element {
	const { t } = useTranslation('dashboard');
	return (
		<SettingsSection
			title={t('section_context_links')}
			icon={<LinkIcon size={14} />}
			defaultOpen={!!contextLinks.linksData.length}
		>
			<div className="context-links-section">
				<ContextLinks
					contextLinks={contextLinks}
					setContextLinks={setContextLinks}
					selectedWidget={selectedWidget}
				/>
			</div>
		</SettingsSection>
	);
}
