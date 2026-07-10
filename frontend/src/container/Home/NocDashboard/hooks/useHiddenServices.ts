import getLocalStorageKey from 'api/browser/localstorage/get';
import setLocalStorageKey from 'api/browser/localstorage/set';
import { LOCALSTORAGE } from 'constants/localStorage';
import { useCallback, useState } from 'react';

export interface UseHiddenServicesResult {
	hidden: Set<string>;
	toggle: (name: string) => void;
}

// 숨긴 서비스 집합을 localStorage에 영속화 — load-generator처럼 관제 무의미한
// 계열을 한 번 숨기면 재방문에도 유지된다. 손상된 저장값은 빈 집합으로 폴백.
function load(): Set<string> {
	try {
		const raw = getLocalStorageKey(LOCALSTORAGE.NOC_TREND_HIDDEN);
		const arr = raw ? JSON.parse(raw) : [];
		return new Set(
			Array.isArray(arr) ? arr.filter((x): x is string => typeof x === 'string') : [],
		);
	} catch {
		return new Set();
	}
}

export default function useHiddenServices(): UseHiddenServicesResult {
	const [hidden, setHidden] = useState<Set<string>>(load);
	const toggle = useCallback((name: string): void => {
		setHidden((prev) => {
			const next = new Set(prev);
			if (next.has(name)) {
				next.delete(name);
			} else {
				next.add(name);
			}
			setLocalStorageKey(LOCALSTORAGE.NOC_TREND_HIDDEN, JSON.stringify([...next]));
			return next;
		});
	}, []);
	return { hidden, toggle };
}
