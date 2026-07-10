import { useCallback, useState } from 'react';

export interface UseHiddenServicesResult {
	hidden: Set<string>;
	toggle: (name: string) => void;
}

// 레인 D(Task 5)가 localStorage 영속화를 얹는다 — 시드는 세션 내 메모리 상태만.
export default function useHiddenServices(): UseHiddenServicesResult {
	const [hidden, setHidden] = useState<Set<string>>(new Set());
	const toggle = useCallback((name: string): void => {
		setHidden((prev) => {
			const next = new Set(prev);
			if (next.has(name)) {
				next.delete(name);
			} else {
				next.add(name);
			}
			return next;
		});
	}, []);
	return { hidden, toggle };
}
