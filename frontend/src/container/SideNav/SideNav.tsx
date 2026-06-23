import {
	MouseEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';
import { useTranslation } from 'react-i18next';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { useLocation } from 'react-router-dom';
import { Button, Dropdown, Tooltip } from 'antd';
import logEvent from 'api/common/logEvent';
import cx from 'classnames';
import { FeatureKeys } from 'constants/features';
import ROUTES from 'constants/routes';
import { GlobalShortcuts } from 'constants/shortcuts/globalShortcuts';
import { useKeyboardHotkeys } from 'hooks/hotkeys/useKeyboardHotkeys';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import history from 'lib/history';
import {
	ArrowUpRight,
	ChevronDown,
	ChevronsDown,
	ChevronUp,
	Cog,
	Ellipsis,
	GitCommitVertical,
	LampDesk,
	PackagePlus,
	ScrollText,
} from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { AppState } from 'store/reducers';
import AppReducer from 'types/reducer/app';
import { USER_ROLES } from 'types/roles';
import { checkVersionState } from 'utils/app';
import { isModifierKeyPressed } from 'utils/app';
import { openInNewTab } from 'utils/navigation';

import ktdsLogoNegativeUrl from '@/assets/Logos/ktds-logo-negative.png';
import ktdsLogoPositiveUrl from '@/assets/Logos/ktds-logo-positive.png';

import { useCmdK } from '../../providers/cmdKProvider';
import { routeConfig } from './config';
import { getQueryString } from './helper';
import {
	defaultMoreMenuItems,
	getHelpSupportDropdownMenuItems,
	helpSupportMenuItem,
	primaryMenuItems,
	settingsNavItemKeyMap,
} from './menuItems';
import { useSettingsNavItems } from './useSettingsNavItems';
import NavItem from './NavItem/NavItem';
import {
	CHANGELOG_LABEL,
	DropdownSeparator,
	SidebarItem,
} from './sideNav.types';
import { getActiveMenuKeyFromPath } from './sideNav.utils';

import './SideNav.styles.scss';

const sideNavItemKeyMap: Record<string, string> = {
	home: 'home',
	alerts: 'alerts',
	dashboards: 'dashboards',
	services: 'services',
	logs: 'logs',
	traces: 'traces',
	metrics: 'metrics',
	infrastructure: 'infrastructure',
	integrations: 'integrations',
	exceptions: 'exceptions',
	'external-apis': 'external_apis',
	'messaging-queues': 'messaging_queues',
	'service-map': 'service_map',
	'meter-explorer': 'cost_meter',
};

// eslint-disable-next-line sonarjs/cognitive-complexity
function SideNav({ isPinned }: { isPinned: boolean }): JSX.Element {
	const { t } = useTranslation(['routes', 'settings', 'helpSupport']);
	const isDarkMode = useIsDarkMode();
	const { openCmdK } = useCmdK();
	const { pathname, search } = useLocation();
	const { currentVersion, latestVersion, isCurrentVersionError } = useSelector<
		AppState,
		AppReducer
	>((state) => state.app);

	const {
		user,
		featureFlags,
		trialInfo,
		isLoggedIn,
		userPreferences,
		changelog,
		toggleChangelogModal,
	} = useAppContext();

	const [helpSupportDropdownMenuItems, setHelpSupportDropdownMenuItems] =
		useState<(SidebarItem | DropdownSeparator)[]>(() =>
			getHelpSupportDropdownMenuItems(t),
		);

	const [hasScroll, setHasScroll] = useState(false);
	const navTopSectionRef = useRef<HTMLDivElement>(null);
	const [isDropdownOpen, setIsDropdownOpen] = useState(false);

	const [isHovered, setIsHovered] = useState(false);
	const [secondaryMenuItems, setSecondaryMenuItems] = useState<SidebarItem[]>(
		[],
	);

	const handleMouseEnter = useCallback(() => {
		setIsHovered(true);
	}, []);

	const handleMouseLeave = useCallback(() => {
		setIsHovered(false);
	}, []);

	const checkScroll = useCallback((): void => {
		if (navTopSectionRef.current) {
			const { scrollHeight, clientHeight, scrollTop } = navTopSectionRef.current;
			const isAtBottom = scrollHeight - clientHeight - scrollTop <= 8;
			setHasScroll(scrollHeight > clientHeight + 24 && !isAtBottom); // 24px - buffer height to show show more
		}
	}, []);

	useEffect(() => {
		checkScroll();
		window.addEventListener('resize', checkScroll);

		// Create a MutationObserver to watch for content changes
		const observer = new MutationObserver(checkScroll);
		const navTopSection = navTopSectionRef.current;

		if (navTopSection) {
			observer.observe(navTopSection, {
				childList: true,
				subtree: true,
				attributes: true,
			});

			// Add scroll event listener
			navTopSection.addEventListener('scroll', checkScroll);
		}

		return (): void => {
			window.removeEventListener('resize', checkScroll);
			observer.disconnect();
			if (navTopSection) {
				navTopSection.removeEventListener('scroll', checkScroll);
			}
		};
	}, [checkScroll]);

	const {
		isCloudUser,
		isEnterpriseSelfHostedUser,
		isCommunityUser,
		isCommunityEnterpriseUser,
	} = useGetTenantLicense();

	const [licenseTag, setLicenseTag] = useState('');
	const isAdmin = user.role === USER_ROLES.ADMIN;
	const isEditor = user.role === USER_ROLES.EDITOR;

	const computedSecondaryMenuItems = useMemo(() => {
		const shouldShowIntegrationsValue =
			(isCloudUser || isEnterpriseSelfHostedUser) && (isAdmin || isEditor);

		return defaultMoreMenuItems.map((item) => ({
			...item,
			isPinned: false,
			isEnabled:
				item.key === ROUTES.INTEGRATIONS
					? shouldShowIntegrationsValue
					: item.isEnabled,
		}));
	}, [isCloudUser, isEnterpriseSelfHostedUser, isAdmin, isEditor]);

	// Track if we've done the initial sync (to avoid overwriting user actions during session)
	const hasInitializedRef = useRef(false);

	// Sync state only on initial load when userPreferences first becomes available
	useEffect(() => {
		// Only sync once: when userPreferences loads for the first time
		if (!hasInitializedRef.current && userPreferences !== null) {
			setSecondaryMenuItems(computedSecondaryMenuItems);
			hasInitializedRef.current = true;
		}
	}, [computedSecondaryMenuItems, userPreferences]);

	const isOnboardingV3Enabled = featureFlags?.find(
		(flag) => flag.name === FeatureKeys.ONBOARDING_V3,
	)?.active;

	const isChatSupportEnabled = featureFlags?.find(
		(flag) => flag.name === FeatureKeys.CHAT_SUPPORT,
	)?.active;

	const isPremiumSupportEnabled = featureFlags?.find(
		(flag) => flag.name === FeatureKeys.PREMIUM_SUPPORT,
	)?.active;

	const isLatestVersion = checkVersionState(currentVersion, latestVersion);

	const [showVersionUpdateNotification, setShowVersionUpdateNotification] =
		useState(false);

	const [isMoreMenuCollapsed, setIsMoreMenuCollapsed] = useState(false);

	const { registerShortcut, deregisterShortcut } = useKeyboardHotkeys();

	const isWorkspaceBlocked = trialInfo?.workSpaceBlock || false;

	const onClickGetStarted = (event: MouseEvent): void => {
		logEvent('Sidebar: Menu clicked', {
			menuRoute: '/get-started',
			menuLabel: 'Get Started',
		});

		const onboaringRoute = isOnboardingV3Enabled
			? ROUTES.GET_STARTED_WITH_CLOUD
			: ROUTES.GET_STARTED;

		if (isModifierKeyPressed(event)) {
			openInNewTab(onboaringRoute);
		} else {
			history.push(onboaringRoute);
		}
	};

	const onClickHandler = useCallback(
		(key: string, event: MouseEvent | null) => {
			const params = new URLSearchParams(search);
			const availableParams = routeConfig[key];

			const queryString = getQueryString(availableParams || [], params);

			if (pathname !== key) {
				if (event && isModifierKeyPressed(event)) {
					openInNewTab(`${key}?${queryString.join('&')}`);
				} else {
					history.push(`${key}?${queryString.join('&')}`, {
						from: pathname,
					});
				}
			}
		},
		[pathname, search],
	);

	const activeMenuKey = useMemo(
		() => getActiveMenuKeyFromPath(pathname),
		[pathname],
	);

	// Settings sub-items live under /settings/* and share the same base path,
	// so the base-path-derived activeMenuKey can't distinguish them. Match on the
	// full pathname instead (with the same sub-route handling the settings page used).
	const isSettingsItemActive = useCallback(
		(item: SidebarItem): boolean => {
			const key = item.key as string;
			if (
				(pathname.startsWith(ROUTES.ALL_CHANNELS) ||
					pathname.startsWith(ROUTES.CHANNELS_EDIT)) &&
				key === ROUTES.ALL_CHANNELS
			) {
				return true;
			}
			if (
				pathname.startsWith(ROUTES.ROLES_SETTINGS) &&
				key === ROUTES.ROLES_SETTINGS
			) {
				return true;
			}
			return pathname === key;
		},
		[pathname],
	);

	const [isSettingsMenuCollapsed, setIsSettingsMenuCollapsed] = useState(true);

	const gatedSettingsSections = useSettingsNavItems();

	useEffect(() => {
		if (isCloudUser) {
			setLicenseTag('Cloud');
		} else if (isEnterpriseSelfHostedUser) {
			setLicenseTag('Enterprise');
		} else if (isCommunityEnterpriseUser) {
			setLicenseTag('Free');
		} else if (isCommunityUser) {
			setLicenseTag('Community');
		}
	}, [
		isCloudUser,
		isEnterpriseSelfHostedUser,
		isCommunityEnterpriseUser,
		isCommunityUser,
	]);

	useEffect(() => {
		if (!isAdmin) {
			setHelpSupportDropdownMenuItems((prevState) =>
				prevState.filter(
					(item) => !('key' in item) || item.key !== 'invite-collaborators',
				),
			);
		}

		const showAddCreditCardModal =
			!isPremiumSupportEnabled && !trialInfo?.trialConvertedToSubscription;

		if (
			!(
				isLoggedIn &&
				isChatSupportEnabled &&
				!showAddCreditCardModal &&
				(isCloudUser || isEnterpriseSelfHostedUser)
			)
		) {
			setHelpSupportDropdownMenuItems((prevState) =>
				prevState.filter((item) => !('key' in item) || item.key !== 'chat-support'),
			);
		}

		if (changelog) {
			const firstTwoFeatures = changelog.features.slice(0, 2);
			const dropdownItems: SidebarItem[] = firstTwoFeatures.map(
				(feature, idx) => ({
					key: `changelog-${idx + 1}`,
					label: (
						<div className="nav-item-label-container">
							<span>{feature.title}</span>
						</div>
					),
					icon: idx === 0 ? <LampDesk size={14} /> : <GitCommitVertical size={14} />,
					itemKey: `changelog-${idx + 1}`,
				}),
			);
			const changelogKey = CHANGELOG_LABEL.toLowerCase().replace(' ', '-');
			setHelpSupportDropdownMenuItems((prevState) => {
				if (dropdownItems.length === 0) {
					return [
						...prevState,
						{
							type: 'divider',
						},
						{
							key: changelogKey,
							label: (
								<div className="nav-item-label-container">
									<span>{t('helpSupport:full_changelog')}</span>
									<ArrowUpRight size={14} />
								</div>
							),
							icon: <ScrollText size={14} />,
							itemKey: changelogKey,
							isExternal: true,
							url: 'https://signoz.io/changelog/',
						},
					];
				}

				return [
					...prevState,
					{
						type: 'divider',
					},
					{
						type: 'group',
						label: t('helpSupport:whats_new'),
					},
					...dropdownItems,
					{
						key: changelogKey,
						label: (
							<div className="nav-item-label-container">
								<span>{t('helpSupport:full_changelog')}</span>
								<ArrowUpRight size={14} />
							</div>
						),
						icon: <ScrollText size={14} />,
						itemKey: changelogKey,
						isExternal: true,
						url: 'https://signoz.io/changelog/',
					},
				];
			});
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [
		isAdmin,
		isChatSupportEnabled,
		isPremiumSupportEnabled,
		isCloudUser,
		trialInfo,
		changelog,
	]);

	const handleMenuItemClick = (event: MouseEvent, item: SidebarItem): void => {
		if (item.key === 'quick-search') {
			openCmdK();
		} else if (item) {
			onClickHandler(item?.key as string, event);
		}
		logEvent('Sidebar V2: Menu clicked', {
			menuRoute: item?.key,
			menuLabel: item?.label,
		});
	};

	useEffect(() => {
		registerShortcut(GlobalShortcuts.NavigateToHome, () =>
			onClickHandler(ROUTES.HOME, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToServices, () =>
			onClickHandler(ROUTES.APPLICATION, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToTraces, () =>
			onClickHandler(ROUTES.TRACES_EXPLORER, null),
		);

		registerShortcut(GlobalShortcuts.NavigateToLogs, () =>
			onClickHandler(ROUTES.LOGS, null),
		);

		registerShortcut(GlobalShortcuts.NavigateToDashboards, () =>
			onClickHandler(ROUTES.ALL_DASHBOARD, null),
		);

		registerShortcut(GlobalShortcuts.NavigateToMessagingQueues, () =>
			onClickHandler(ROUTES.MESSAGING_QUEUES_OVERVIEW, null),
		);

		registerShortcut(GlobalShortcuts.NavigateToAlerts, () =>
			onClickHandler(ROUTES.LIST_ALL_ALERT, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToExceptions, () =>
			onClickHandler(ROUTES.ALL_ERROR, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToTracesFunnel, () =>
			onClickHandler(ROUTES.TRACES_FUNNELS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToTracesViews, () =>
			onClickHandler(ROUTES.TRACES_SAVE_VIEWS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToMetricsSummary, () =>
			onClickHandler(ROUTES.METRICS_EXPLORER, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToMetricsExplorer, () =>
			onClickHandler(ROUTES.METRICS_EXPLORER_EXPLORER, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToMetricsViews, () =>
			onClickHandler(ROUTES.METRICS_EXPLORER_VIEWS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettings, () =>
			onClickHandler(ROUTES.SETTINGS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettingsIngestion, () =>
			onClickHandler(ROUTES.INGESTION_SETTINGS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettingsBilling, () =>
			onClickHandler(ROUTES.BILLING, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettingsNotificationChannels, () =>
			onClickHandler(ROUTES.ALL_CHANNELS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettingsServiceAccounts, () =>
			onClickHandler(ROUTES.SERVICE_ACCOUNTS_SETTINGS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettingsRoles, () =>
			onClickHandler(ROUTES.ROLES_SETTINGS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToSettingsMembers, () =>
			onClickHandler(ROUTES.MEMBERS_SETTINGS, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToLogsPipelines, () =>
			onClickHandler(ROUTES.LOGS_PIPELINES, null),
		);
		registerShortcut(GlobalShortcuts.NavigateToLogsViews, () =>
			onClickHandler(ROUTES.LOGS_SAVE_VIEWS, null),
		);
		return (): void => {
			deregisterShortcut(GlobalShortcuts.NavigateToHome);
			deregisterShortcut(GlobalShortcuts.NavigateToServices);
			deregisterShortcut(GlobalShortcuts.NavigateToTraces);
			deregisterShortcut(GlobalShortcuts.NavigateToLogs);
			deregisterShortcut(GlobalShortcuts.NavigateToDashboards);
			deregisterShortcut(GlobalShortcuts.NavigateToAlerts);
			deregisterShortcut(GlobalShortcuts.NavigateToExceptions);
			deregisterShortcut(GlobalShortcuts.NavigateToMessagingQueues);
			deregisterShortcut(GlobalShortcuts.NavigateToTracesFunnel);
			deregisterShortcut(GlobalShortcuts.NavigateToMetricsSummary);
			deregisterShortcut(GlobalShortcuts.NavigateToMetricsExplorer);
			deregisterShortcut(GlobalShortcuts.NavigateToMetricsViews);
			deregisterShortcut(GlobalShortcuts.NavigateToSettings);
			deregisterShortcut(GlobalShortcuts.NavigateToSettingsIngestion);
			deregisterShortcut(GlobalShortcuts.NavigateToSettingsBilling);
			deregisterShortcut(GlobalShortcuts.NavigateToSettingsNotificationChannels);
			deregisterShortcut(GlobalShortcuts.NavigateToSettingsServiceAccounts);
			deregisterShortcut(GlobalShortcuts.NavigateToSettingsRoles);
			deregisterShortcut(GlobalShortcuts.NavigateToSettingsMembers);
			deregisterShortcut(GlobalShortcuts.NavigateToLogsPipelines);
			deregisterShortcut(GlobalShortcuts.NavigateToLogsViews);
			deregisterShortcut(GlobalShortcuts.NavigateToTracesViews);
		};
	}, [deregisterShortcut, onClickHandler, registerShortcut]);

	const moreMenuItems = useMemo(
		() => secondaryMenuItems.filter((i) => i.isEnabled),
		[secondaryMenuItems],
	);

	// Get active "More" items that should be visible in collapsed state
	const activeMoreMenuItems = useMemo(
		() => moreMenuItems.filter((item) => activeMenuKey === item.key),
		[moreMenuItems, activeMenuKey],
	);

	// Check if sidebar is collapsed (not pinned, not hovered, and no dropdown open)
	const isCollapsed = !isPinned && !isHovered && !isDropdownOpen;

	const renderNavItems = (
		items: SidebarItem[],
		getIsActive?: (item: SidebarItem) => boolean,
	): JSX.Element => (
		<>
			{items.map((item, index) => {
				const translatedLabel =
					item.itemKey && sideNavItemKeyMap[item.itemKey as string]
						? t(sideNavItemKeyMap[item.itemKey as string])
						: item.label;
				return (
					<NavItem
						showIcon
						key={item.key || index}
						item={{ ...item, label: translatedLabel }}
						isActive={
							getIsActive ? getIsActive(item) : activeMenuKey === item.key
						}
						isDisabled={
							isWorkspaceBlocked &&
							item.key !== ROUTES.BILLING &&
							item.key !== ROUTES.SETTINGS
						}
						onClick={(event): void => {
							handleMenuItemClick(event, item);
						}}
					/>
				);
			})}
		</>
	);

	// Check scroll when menu items change
	useEffect(() => {
		checkScroll();
	}, [checkScroll, moreMenuItems]);

	const handleScrollForMore = (): void => {
		if (navTopSectionRef.current) {
			navTopSectionRef.current.scrollTo({
				top: navTopSectionRef.current.scrollHeight,
				behavior: 'smooth',
			});
		}
	};

	// eslint-disable-next-line sonarjs/cognitive-complexity
	const handleHelpSupportMenuItemClick = (info: SidebarItem): void => {
		const item = helpSupportDropdownMenuItems.find(
			(item) => !('type' in item) && item.key === info.key,
		);

		if (item && !('type' in item) && item.isExternal && item.url) {
			openInNewTab(item.url);
		}

		const event = (info as SidebarItem & { domEvent?: MouseEvent }).domEvent;

		if (item && !('type' in item)) {
			logEvent('Help Popover: Item clicked', {
				menuRoute: item.key,
				menuLabel: String(item.label),
			});

			switch (item.key) {
				case ROUTES.SHORTCUTS:
					if (event && isModifierKeyPressed(event)) {
						openInNewTab(ROUTES.SHORTCUTS);
					} else {
						history.push(ROUTES.SHORTCUTS);
					}
					break;
				case 'invite-collaborators':
					if (event && isModifierKeyPressed(event)) {
						openInNewTab(`${ROUTES.ORG_SETTINGS}#invite-team-members`);
					} else {
						history.push(`${ROUTES.ORG_SETTINGS}#invite-team-members`);
					}
					break;
				case 'chat-support':
					if (window.pylon) {
						window.Pylon('show');
					}
					break;
				case 'changelog-1':
				case 'changelog-2':
					toggleChangelogModal();
					break;
				default:
					break;
			}
		}
	};

	const onClickVersionHandler = useCallback((): void => {
		if (!changelog) {
			return;
		}

		toggleChangelogModal();
	}, [changelog, toggleChangelogModal]);

	useEffect(() => {
		if (!isLatestVersion && !isCloudUser) {
			setShowVersionUpdateNotification(true);
		} else {
			setShowVersionUpdateNotification(false);
		}
	}, [
		currentVersion,
		latestVersion,
		isCurrentVersionError,
		isLatestVersion,
		isCloudUser,
		isEnterpriseSelfHostedUser,
	]);

	return (
		<div className={cx('sidenav-container', isPinned && 'pinned')}>
			<div
				className={cx(
					'sideNav',
					isPinned && 'pinned',
					isDropdownOpen && 'dropdown-open',
				)}
				onMouseEnter={handleMouseEnter}
				onMouseLeave={handleMouseLeave}
			>
				<div className="brand-container">
					<div className="brand">
						<div className="brand-company-meta">
							<div
								className="brand-logo"
								onClick={(event: MouseEvent): void => {
									// Current home page
									onClickHandler(ROUTES.HOME, event);
								}}
							>
								<img
									src={isDarkMode ? ktdsLogoNegativeUrl : ktdsLogoPositiveUrl}
									alt="kt ds"
								/>
								<span className="brand-logo-name">DS-APM</span>
							</div>

							{licenseTag && (
								<div
									className={cx(
										'brand-title-section',
										isCommunityEnterpriseUser && 'community-enterprise-user',
										isCloudUser && 'cloud-user',
										showVersionUpdateNotification &&
											changelog &&
											'version-update-notification',
									)}
								>
									<span className="license-type"> {licenseTag} </span>

									{currentVersion && (
										<Tooltip
											placement="bottomLeft"
											overlayClassName="version-tooltip-overlay"
											arrow={false}
											overlay={
												showVersionUpdateNotification &&
												changelog && (
													<div className="version-update-notification-tooltip">
														<div className="version-update-notification-tooltip-title">
															There&apos;s a new version available.
														</div>

														<div className="version-update-notification-tooltip-content">
															{latestVersion}
														</div>
													</div>
												)
											}
										>
											<div className="version-container">
												<span
													className={cx('version', changelog && 'version-clickable')}
													onClick={onClickVersionHandler}
												>
													{currentVersion}
												</span>

												{showVersionUpdateNotification && changelog && (
													<span className="version-update-notification-dot-icon" />
												)}
											</div>
										</Tooltip>
									)}
								</div>
							)}
						</div>
					</div>
				</div>

				<div
					className={cx(
						`nav-wrapper`,
						isCloudUser && 'nav-wrapper-cloud',
						hasScroll && 'scroll-available',
					)}
				>
					<div className={cx('nav-top-section')} ref={navTopSectionRef}>
						{isCloudUser && user?.role !== USER_ROLES.VIEWER && (
							<div className="get-started-nav-items">
								<Button
									className="get-started-btn"
									disabled={isWorkspaceBlocked}
									onClick={(event: MouseEvent): void => {
										if (isWorkspaceBlocked) {
											return;
										}
										onClickGetStarted(event);
									}}
								>
									<PackagePlus size={16} />
									<div className="license tag nav-item-label"> New source </div>
								</Button>
							</div>
						)}

						<div className="primary-nav-items">
							{renderNavItems(primaryMenuItems)}
						</div>

						{moreMenuItems.length > 0 && (
							<div
								className={cx(
									'more-nav-items',
									isMoreMenuCollapsed ? 'collapsed' : 'expanded',
									isCollapsed && 'sidebar-collapsed',
								)}
							>
								{!isCollapsed && (
									<div className="nav-title-section">
										<div
											className="nav-section-title"
											onClick={(): void => {
												// Only allow toggling when sidebar is open (pinned, hovered, or dropdown open)
												if (isCollapsed) {
													return;
												}
												const newCollapsedState = !isMoreMenuCollapsed;
												logEvent('Sidebar V2: More menu clicked', {
													action: isMoreMenuCollapsed ? 'expand' : 'collapse',
												});
												setIsMoreMenuCollapsed(newCollapsedState);
											}}
										>
											<div className="nav-section-title-icon">
												<Ellipsis size={16} />
											</div>

											<div className="nav-section-title-text">MORE</div>

											<div className="collapse-expand-section-icon">
												{isMoreMenuCollapsed ? (
													<ChevronDown size={16} />
												) : (
													<ChevronUp size={16} />
												)}
											</div>
										</div>
									</div>
								)}

								<div className="nav-items-section">
									{/* Show all items when expanded, only active items when collapsed */}
									{isCollapsed
										? renderNavItems(activeMoreMenuItems)
										: renderNavItems(moreMenuItems)}
								</div>
							</div>
						)}

						{!isCollapsed && (
							<div className={cx('more-nav-items', 'settings-nav-items')}>
								<div className="nav-title-section">
									<div
										className="nav-section-title"
										data-testid="settings-group-header"
										onClick={(): void => {
											setIsSettingsMenuCollapsed(false);
											history.push(ROUTES.MY_SETTINGS);
											logEvent('Sidebar V2: Settings group clicked', {});
										}}
									>
										<div className="nav-section-title-icon">
											<Cog size={16} />
										</div>
										<div className="nav-section-title-text">
											{t('routes:settings_title')}
										</div>
										<div
											className="collapse-expand-section-icon"
											onClick={(e): void => {
												e.stopPropagation();
												setIsSettingsMenuCollapsed((v) => !v);
											}}
										>
											{isSettingsMenuCollapsed ? (
												<ChevronDown size={16} />
											) : (
												<ChevronUp size={16} />
											)}
										</div>
									</div>
								</div>

								{!isSettingsMenuCollapsed && (
									<div className="nav-items-section">
										{renderNavItems(
											gatedSettingsSections
												.flatMap((section) => section.items)
												.filter((i) => i.isEnabled)
												.map((i) => ({
													...i,
													label: t(
														settingsNavItemKeyMap[i.itemKey as string] ??
															(i.label as string),
													),
												})),
											isSettingsItemActive,
										)}
									</div>
								)}
							</div>
						)}

						<div className="scroll-for-more-container">
							<div className="scroll-for-more" onClick={handleScrollForMore}>
								<div className="scroll-for-more-icon">
									<ChevronsDown size={16} />
								</div>

								<div className="scroll-for-more-label">Scroll for more</div>
							</div>
						</div>
					</div>

					<div className="nav-bottom-section">
						<div className="secondary-nav-items">
							<div className="nav-dropdown-item">
								<Dropdown
									menu={{
										items: helpSupportDropdownMenuItems,
										onClick: handleHelpSupportMenuItemClick,
									}}
									placement="topLeft"
									overlayClassName="nav-dropdown-overlay help-support-dropdown"
									trigger={['click']}
									onOpenChange={(open): void => setIsDropdownOpen(open)}
								>
									<div className="nav-item">
										<div className="nav-item-data" data-testid="help-support-nav-item">
											<div className="nav-item-icon">{helpSupportMenuItem.icon}</div>

											<div className="nav-item-label">
												{t('helpSupport:help_support')}
											</div>
										</div>
									</div>
								</Dropdown>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}

export default SideNav;
