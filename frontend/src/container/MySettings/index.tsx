import { useEffect, useState } from 'react';
import { useMutation } from 'react-query';
import { useTranslation } from 'react-i18next';
import { Modal, Radio, RadioChangeEvent, Switch, Tag } from 'antd';
import setLocalStorageApi from 'api/browser/localstorage/set';
import logEvent from 'api/common/logEvent';
import updateUserPreference from 'api/v1/user/preferences/name/update';
import i18n from 'i18next';
import { AxiosError } from 'axios';
import { USER_PREFERENCES } from 'constants/userPreferences';
import useThemeMode, { useIsDarkMode, useSystemTheme } from 'hooks/useDarkMode';
import { useNotifications } from 'hooks/useNotifications';
import { MonitorCog, Moon, Sun } from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { UserPreference } from 'types/api/preferences/preference';
import { showErrorNotification } from 'utils/error';

import LicenseSection from './LicenseSection';
import TimezoneAdaptation from './TimezoneAdaptation/TimezoneAdaptation';
import UserInfo from './UserInfo';

import './MySettings.styles.scss';

function MySettings(): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { userPreferences, updateUserPreferenceInContext } = useAppContext();
	const { toggleTheme, autoSwitch, setAutoSwitch } = useThemeMode();
	const systemTheme = useSystemTheme();
	const { notifications } = useNotifications();
	const { t } = useTranslation(['settings']);

	const [sideNavPinned, setSideNavPinned] = useState(false);
	const [language, setLanguage] = useState<'en' | 'ko'>(
		i18n.language?.startsWith('ko') ? 'ko' : 'en',
	);

	useEffect(() => {
		if (userPreferences) {
			setSideNavPinned(
				userPreferences.find(
					(preference) => preference.name === USER_PREFERENCES.SIDENAV_PINNED,
				)?.value as boolean,
			);
		}
	}, [userPreferences]);

	const {
		mutate: updateUserPreferenceMutation,
		isLoading: isUpdatingUserPreference,
	} = useMutation(updateUserPreference, {
		onSuccess: () => {
			// No need to do anything on success since we've already updated the state optimistically
		},
		onError: (error) => {
			showErrorNotification(notifications, error as AxiosError);
		},
	});

	const themeOptions = [
		{
			label: (
				<div className="theme-option">
					<Moon data-testid="dark-theme-icon" size={12} /> Dark{' '}
				</div>
			),
			value: 'dark',
		},
		{
			label: (
				<div className="theme-option">
					<Sun size={12} data-testid="light-theme-icon" /> Light{' '}
					<Tag bordered={false} color="geekblue">
						Beta
					</Tag>
				</div>
			),
			value: 'light',
		},
		{
			label: (
				<div className="theme-option">
					<MonitorCog size={12} data-testid="auto-theme-icon" /> System{' '}
				</div>
			),
			value: 'auto',
		},
	];

	const [theme, setTheme] = useState(() => {
		if (autoSwitch) {
			return 'auto';
		}
		return isDarkMode ? 'dark' : 'light';
	});

	const handleThemeChange = ({ target: { value } }: RadioChangeEvent): void => {
		logEvent('Account Settings: Theme Changed', {
			theme: value,
		});
		setTheme(value);

		if (value === 'auto') {
			setAutoSwitch(true);
		} else {
			setAutoSwitch(false);
			// Only toggle if the current theme is different from the target
			const targetIsDark = value === 'dark';
			if (targetIsDark !== isDarkMode) {
				toggleTheme();
			}
		}
	};

	useEffect(() => {
		if (autoSwitch) {
			setTheme('auto');
			return;
		}

		if (isDarkMode) {
			setTheme('dark');
		} else {
			setTheme('light');
		}
	}, [autoSwitch, isDarkMode]);

	const handleLanguageToggle = (checked: boolean): void => {
		const targetLang = checked ? 'ko' : 'en';
		Modal.confirm({
			title: '언어 변경',
			content: checked
				? '설정 메뉴를 한국어로 변환하시겠습니까?'
				: '설정 메뉴를 영어로 변환하시겠습니까?',
			okText: '확인',
			cancelText: '취소',
			className: 'language-change-modal',
			onOk: () => {
				i18n.changeLanguage(targetLang);
				setLanguage(targetLang);
			},
		});
	};

	const handleSideNavPinnedChange = (checked: boolean): void => {
		logEvent('Account Settings: Sidebar Pinned Changed', {
			pinned: checked,
		});
		// Optimistically update the UI
		setSideNavPinned(checked);

		// Save to localStorage immediately for instant feedback
		setLocalStorageApi(USER_PREFERENCES.SIDENAV_PINNED, checked.toString());

		// Update the context immediately
		const save = {
			name: USER_PREFERENCES.SIDENAV_PINNED,
			value: checked,
		};
		updateUserPreferenceInContext(save as UserPreference);

		// Make the API call in the background
		updateUserPreferenceMutation(
			{
				name: USER_PREFERENCES.SIDENAV_PINNED,
				value: checked,
			},
			{
				onError: (error) => {
					// Revert the state if the API call fails
					setSideNavPinned(!checked);
					updateUserPreferenceInContext({
						name: USER_PREFERENCES.SIDENAV_PINNED,
						value: !checked,
					} as UserPreference);
					// Also revert localStorage
					setLocalStorageApi(USER_PREFERENCES.SIDENAV_PINNED, (!checked).toString());
					showErrorNotification(notifications, error as AxiosError);
				},
			},
		);
	};

	return (
		<div className="my-settings-container">
			<div className="user-info-section">
				<div className="user-info-section-header">
					<div className="user-info-section-title">{t('settings:account_title')}</div>

					<div className="user-info-section-subtitle">
						{t('settings:account_subtitle')}
					</div>
				</div>

				<div className="user-info-container">
					<UserInfo />
				</div>
			</div>

			<div className="user-preference-section">
				<div className="user-preference-section-header">
					<div className="user-preference-section-title">{t('settings:preferences_title')}</div>

					<div className="user-preference-section-subtitle">
						{t('settings:preferences_subtitle')}
					</div>
				</div>

				<div className="user-preference-section-content">
					<div className="user-preference-section-content-item theme-selector">
						<div className="user-preference-section-content-item-title-action">
							{t('settings:theme_title')}
							<Radio.Group
								options={themeOptions}
								onChange={handleThemeChange}
								value={theme}
								optionType="button"
								buttonStyle="solid"
								data-testid="theme-selector"
								size="middle"
							/>
						</div>

						<div className="user-preference-section-content-item-description">
							{t('settings:theme_description')}
						</div>

						{autoSwitch && (
							<div className="auto-theme-info">
								<div className="auto-theme-status">
									{t('settings:auto_theme_status')}{' '}
									<strong>{systemTheme === 'dark' ? 'Dark' : 'Light'}</strong>
								</div>
							</div>
						)}
					</div>

					<TimezoneAdaptation />

					<div className="user-preference-section-content-item">
						<div className="user-preference-section-content-item-title-action">
							{t('settings:language_title')}{' '}
							<Switch
								checked={language === 'ko'}
								onChange={handleLanguageToggle}
								checkedChildren="한국어"
								unCheckedChildren="English"
								data-testid="language-toggle-switch"
							/>
						</div>
						<div className="user-preference-section-content-item-description">
							{t('settings:language_description')}
						</div>
					</div>

					<div className="user-preference-section-content-item">
						<div className="user-preference-section-content-item-title-action">
							{t('settings:sidenav_title')}{' '}
							<Switch
								checked={sideNavPinned}
								onChange={handleSideNavPinnedChange}
								loading={isUpdatingUserPreference}
								data-testid="side-nav-pinned-switch"
							/>
						</div>

						<div className="user-preference-section-content-item-description">
							{t('settings:sidenav_description')}
						</div>
					</div>
				</div>
			</div>

			<LicenseSection />
		</div>
	);
}

export default MySettings;
