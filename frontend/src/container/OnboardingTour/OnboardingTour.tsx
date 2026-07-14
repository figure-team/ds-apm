import './OnboardingTour.styles.scss';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Tour, TourProps } from 'antd';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import { LOCALSTORAGE } from 'constants/localStorage';

/** SideNav가 구독해 투어 동안 사이드바를 펼침 상태로 고정하는 이벤트 */
export const TOUR_ACTIVE_EVENT = 'TOUR_ACTIVE';

const NAV_HOME_SELECTOR = '[data-testid="home"]';
const NAV_ALERTS_SELECTOR = '[data-testid="alerts"]';
const NAV_DASHBOARDS_SELECTOR = '[data-testid="dashboards"]';
// 설정 블록도 more-nav-items 클래스를 공유하므로 :not으로 구분한다
const NAV_MORE_SELECTOR = '.more-nav-items:not(.settings-nav-items)';
const NAV_SETTINGS_SELECTOR = '.settings-nav-items';

// rc-tour(antd 5.11)는 center placement 정렬 정의가 없어 타겟 없는 스텝이
// 정중앙에 놓이지 않는다 — step className이 팝업 래퍼에 부착되는 점을 이용해
// CSS(position:fixed + translate)로 뷰포트 정중앙에 고정한다
const CENTER_STEP_CLASS = 'onboarding-tour-step-center';

const ANCHOR_POLL_INTERVAL_MS = 300;
const ANCHOR_POLL_MAX_ATTEMPTS = 20;
// MORE·설정 섹션은 사이드바가 펼쳐진 뒤에야 렌더되므로,
// TOUR_ACTIVE 발행 → 펼침 렌더 완료 후에 투어를 연다
const SIDENAV_EXPAND_DELAY_MS = 350;

function getTarget(selector: string): () => HTMLElement {
	// 타겟이 없으면 antd Tour가 중앙 표시로 폴백한다
	return (): HTMLElement => document.querySelector(selector) as HTMLElement;
}

export default function OnboardingTour(): JSX.Element {
	const { t } = useTranslation('tour');
	const [open, setOpen] = useState(false);

	useEffect(() => {
		if (getLocalStorageApi(LOCALSTORAGE.ONBOARDING_TOUR_DONE) === 'true') {
			return undefined;
		}

		let attempts = 0;
		let expandTimer: ReturnType<typeof setTimeout>;

		const poll = setInterval(() => {
			attempts += 1;

			if (document.querySelector(NAV_HOME_SELECTOR)) {
				clearInterval(poll);
				window.dispatchEvent(new CustomEvent(TOUR_ACTIVE_EVENT, { detail: true }));
				expandTimer = setTimeout(() => setOpen(true), SIDENAV_EXPAND_DELAY_MS);
			} else if (attempts >= ANCHOR_POLL_MAX_ATTEMPTS) {
				clearInterval(poll);
			}
		}, ANCHOR_POLL_INTERVAL_MS);

		return (): void => {
			clearInterval(poll);
			clearTimeout(expandTimer);
		};
	}, []);

	const markDone = useCallback((): void => {
		setOpen(false);
		setLocalStorageApi(LOCALSTORAGE.ONBOARDING_TOUR_DONE, 'true');
		window.dispatchEvent(new CustomEvent(TOUR_ACTIVE_EVENT, { detail: false }));
	}, []);

	const steps: TourProps['steps'] = useMemo(() => {
		const nextButtonProps = { children: t('next') };
		const prevButtonProps = { children: t('prev') };

		return [
			{
				title: t('welcome_title'),
				description: t('welcome_desc'),
				className: CENTER_STEP_CLASS,
				nextButtonProps,
			},
			{
				title: t('home_title'),
				description: t('home_desc'),
				target: getTarget(NAV_HOME_SELECTOR),
				nextButtonProps,
				prevButtonProps,
			},
			{
				title: t('alerts_title'),
				description: t('alerts_desc'),
				target: getTarget(NAV_ALERTS_SELECTOR),
				nextButtonProps,
				prevButtonProps,
			},
			{
				title: t('dashboards_title'),
				description: t('dashboards_desc'),
				target: getTarget(NAV_DASHBOARDS_SELECTOR),
				nextButtonProps,
				prevButtonProps,
			},
			{
				title: t('more_title'),
				description: t('more_desc'),
				target: getTarget(NAV_MORE_SELECTOR),
				nextButtonProps,
				prevButtonProps,
			},
			{
				title: t('settings_title'),
				description: t('settings_desc'),
				target: getTarget(NAV_SETTINGS_SELECTOR),
				nextButtonProps,
				prevButtonProps,
			},
			{
				title: t('finish_title'),
				description: t('finish_desc'),
				className: CENTER_STEP_CLASS,
				nextButtonProps: { children: t('confirm') },
				prevButtonProps,
			},
		];
	}, [t]);

	return (
		<Tour
			open={open}
			steps={steps}
			onClose={markDone}
			onFinish={markDone}
			rootClassName="onboarding-tour"
		/>
	);
}
