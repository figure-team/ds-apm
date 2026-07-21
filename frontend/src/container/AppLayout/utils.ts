import ROUTES from 'constants/routes';
import { matchPath } from 'react-router-dom';

export function getRouteKey(pathname: string): string {
	// 1순위: 정확 일치 — 파라미터 패턴이 정적 경로를 가로채지 않도록 먼저 본다.
	const exact = Object.entries(ROUTES).find(([, value]) => value === pathname);
	if (exact) {
		return exact[0];
	}

	// 2순위: 동적 라우트(/services/:servicename 등) 패턴 매칭.
	// 여러 패턴이 겹치면 ROUTES 정의 순서 우선.
	const matched = Object.entries(ROUTES).find(([, value]) =>
		matchPath(pathname, { path: value, exact: true, strict: false }),
	);
	if (matched) {
		return matched[0];
	}

	// 3순위: ROUTES에 없는 설정 하위 경로(/settings/general 등)는 설정 타이틀로 폴백.
	if (pathname.startsWith('/settings')) {
		return 'SETTINGS';
	}

	return 'DEFAULT';
}
