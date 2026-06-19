import { useTranslation } from 'react-i18next';
import awsDarkLogoUrl from '@/assets/Logos/aws-dark.svg';

import AccountActions from './components/AccountActions';

import './HeroSection.style.scss';

function HeroSection(): JSX.Element {
	const { t } = useTranslation('integrations');
	return (
		<div className="hero-section">
			<div className="hero-section__details">
				<div className="hero-section__details-header">
					<div className="hero-section__icon">
						<img src={awsDarkLogoUrl} alt={t('hero.aws')} />
					</div>

					<div className="hero-section__details-title">{t('hero.aws')}</div>
				</div>
				<div className="hero-section__details-description">
					{t('hero.aws_description')}
				</div>
			</div>
			<AccountActions />
		</div>
	);
}

export default HeroSection;
