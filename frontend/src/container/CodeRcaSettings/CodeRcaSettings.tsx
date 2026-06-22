import { useTranslation } from 'react-i18next';
import { Tabs } from 'antd';
import { useAppContext } from 'providers/App/App';
import { useHistory, useLocation } from 'react-router-dom';
import { USER_ROLES } from 'types/roles';

import ConfigTab from './ConfigTab';
import RunsTab from './RunsTab';

import './CodeRcaSettings.styles.scss';

const VALID_TABS = ['config', 'runs'];

function CodeRcaSettings(): JSX.Element {
	const { t } = useTranslation(['codeRca']);
	const { user } = useAppContext();
	const isAdmin = user.role === USER_ROLES.ADMIN;

	const history = useHistory();
	const { search } = useLocation();
	const tabParam = new URLSearchParams(search).get('tab');
	const activeTab = tabParam && VALID_TABS.includes(tabParam) ? tabParam : 'config';

	const handleTabChange = (key: string): void => {
		history.push({ search: `?tab=${key}` });
	};

	return (
		<div
			className="code-rca-settings settings-shell settings-shell--narrow"
			data-testid="code-rca-settings"
		>
			<header className="code-rca-settings__header">
				<h1 className="code-rca-settings__header-title">{t('header_title')}</h1>
				<p className="code-rca-settings__header-subtitle">{t('header_subtitle')}</p>
			</header>
			<Tabs
				activeKey={activeTab}
				onChange={handleTabChange}
				items={[
					{
						key: 'config',
						label: t('tab_config'),
						children: <ConfigTab isAdmin={isAdmin} />,
					},
					{
						key: 'runs',
						label: t('tab_runs'),
						children: <RunsTab />,
					},
				]}
			/>
		</div>
	);
}

export default CodeRcaSettings;
