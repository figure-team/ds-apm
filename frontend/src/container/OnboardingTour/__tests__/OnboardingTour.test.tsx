import { LOCALSTORAGE } from 'constants/localStorage';
import { act, fireEvent, render, screen } from 'tests/test-utils';

import OnboardingTour, { TOUR_ACTIVE_EVENT } from '../OnboardingTour';

// 앵커 폴링(300ms) + 사이드바 펼침 대기(350ms)를 넘기는 시간
const OPEN_DELAY_MS = 700;

describe('OnboardingTour', () => {
	let anchor: HTMLDivElement;

	beforeEach(() => {
		jest.useFakeTimers();
		localStorage.clear();
		anchor = document.createElement('div');
		anchor.setAttribute('data-testid', 'home');
		document.body.appendChild(anchor);
	});

	afterEach(() => {
		jest.useRealTimers();
		anchor.remove();
	});

	it('첫 방문이면 TOUR_ACTIVE 발행 후 투어를 연다', () => {
		const received: boolean[] = [];
		const listener = (event: Event): void => {
			received.push(Boolean((event as CustomEvent).detail));
		};
		window.addEventListener(TOUR_ACTIVE_EVENT, listener);

		render(<OnboardingTour />);

		act(() => {
			jest.advanceTimersByTime(OPEN_DELAY_MS);
		});

		expect(received).toEqual([true]);
		expect(screen.getByText('welcome_title')).toBeInTheDocument();
		// 중앙 고정 CSS 훅(step className)이 팝업 래퍼에 부착되는지 확인
		expect(document.querySelector('.onboarding-tour-step-center')).not.toBeNull();

		window.removeEventListener(TOUR_ACTIVE_EVENT, listener);
	});

	it('완료 플래그가 있으면 열리지 않는다', () => {
		localStorage.setItem(LOCALSTORAGE.ONBOARDING_TOUR_DONE, 'true');

		render(<OnboardingTour />);

		act(() => {
			jest.advanceTimersByTime(OPEN_DELAY_MS * 3);
		});

		expect(screen.queryByText('welcome_title')).not.toBeInTheDocument();
	});

	it('닫으면 완료 플래그를 저장하고 투어를 내린다', () => {
		render(<OnboardingTour />);

		act(() => {
			jest.advanceTimersByTime(OPEN_DELAY_MS);
		});

		const closeButton = document.querySelector('.ant-tour-close');
		expect(closeButton).not.toBeNull();

		fireEvent.click(closeButton as Element);

		expect(localStorage.getItem(LOCALSTORAGE.ONBOARDING_TOUR_DONE)).toBe('true');
		expect(screen.queryByText('welcome_title')).not.toBeInTheDocument();
	});
});
