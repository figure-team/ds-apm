import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Search } from '@signozhq/icons';

function MembersTab(): JSX.Element {
	const { t } = useTranslation(['roles']);
	const [searchQuery, setSearchQuery] = useState('');

	return (
		<div className="role-details-members">
			<div className="role-details-members-search">
				<Search size={12} className="role-details-members-search-icon" />
				<input
					type="text"
					className="role-details-members-search-input"
					placeholder={t('search_add_members_placeholder')}
					value={searchQuery}
					onChange={(e): void => setSearchQuery(e.target.value)}
				/>
			</div>

			{/* Todo: Right now we are only adding the empty state in this cut */}
			<div className="role-details-members-content">
				<div className="role-details-members-empty-state">
					<span
						className="role-details-members-empty-emoji"
						role="img"
						aria-label={t('aria_monocle_face')}
					>
						🧐
					</span>
					<p className="role-details-members-empty-text">
						<span className="role-details-members-empty-text--bold">
							{t('no_members')}
						</span>{' '}
						<span className="role-details-members-empty-text--muted">
							{t('start_adding_members')}
						</span>
					</p>
				</div>
			</div>
		</div>
	);
}

export default MembersTab;
