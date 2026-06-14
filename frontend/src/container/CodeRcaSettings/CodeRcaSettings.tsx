import { useTranslation } from 'react-i18next';
import { Tabs } from 'antd';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import ConfigTab from './ConfigTab';
import RunsTab from './RunsTab';

import './CodeRcaSettings.styles.scss';

function CodeRcaSettings(): JSX.Element {
	const { t } = useTranslation(['codeRca']);
	const { user } = useAppContext();
	const isAdmin = user.role === USER_ROLES.ADMIN;

	return (
		<div className="code-rca-settings" data-testid="code-rca-settings">
			<header className="code-rca-settings__header">
				<h1 className="code-rca-settings__header-title">{t('header_title')}</h1>
				<p className="code-rca-settings__header-subtitle">{t('header_subtitle')}</p>
			</header>
			<Tabs
				defaultActiveKey="config"
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
