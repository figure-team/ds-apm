import { Typography } from 'antd';
import get from 'api/browser/localstorage/get';
import { LOCALSTORAGE } from 'constants/localStorage';
import { THEME_MODE } from 'hooks/useDarkMode/constant';

import ktdsLogoNegativeUrl from '@/assets/Logos/ktds-logo-negative.png';
import ktdsLogoPositiveUrl from '@/assets/Logos/ktds-logo-positive.png';

import './AppLoading.styles.scss';

function AppLoading(): JSX.Element {
	// Get theme from localStorage directly to avoid context dependency
	const getThemeFromStorage = (): boolean => {
		try {
			const theme = get(LOCALSTORAGE.THEME);
			return theme !== THEME_MODE.LIGHT; // Return true for dark, false for light
		} catch (error) {
			// If localStorage is not available, default to dark theme
			return true;
		}
	};

	const isDarkMode = getThemeFromStorage();

	return (
		<div className={`app-loading-container ${isDarkMode ? 'dark' : 'lightMode'}`}>
			<div className="perilin-bg" />
			<div className="app-loading-content">
				<div className="brand">
					<img
						src={isDarkMode ? ktdsLogoNegativeUrl : ktdsLogoPositiveUrl}
						alt="kt ds"
						className="brand-logo"
					/>

					<Typography.Title level={2} className="brand-title">
						DS-APM
					</Typography.Title>
				</div>

				<div className="loader" />
			</div>
		</div>
	);
}

export default AppLoading;
