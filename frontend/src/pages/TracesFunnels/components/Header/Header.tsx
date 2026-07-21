import { useTranslation } from 'react-i18next';

function Header(): JSX.Element {
	const { t } = useTranslation('trace');
	return (
		<div className="traces-funnels-header">
			<div className="traces-funnels-header-title">{t('tab_funnels')}</div>
			<div className="traces-funnels-header-subtitle">
				{t('funnels_subtitle')}
			</div>
		</div>
	);
}

export default Header;
