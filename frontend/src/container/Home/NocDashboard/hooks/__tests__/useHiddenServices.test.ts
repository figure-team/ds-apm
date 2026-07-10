import { act, renderHook } from '@testing-library/react';
import { LOCALSTORAGE } from 'constants/localStorage';

import useHiddenServices from '../useHiddenServices';

const store = new Map<string, string>();
const getMock = jest.fn((key: string): string | null => store.get(key) ?? null);
const setMock = jest.fn((key: string, value: string): boolean => {
	store.set(key, value);
	return true;
});

jest.mock('api/browser/localstorage/get', () => ({
	__esModule: true,
	default: (key: string): string | null => getMock(key),
}));
jest.mock('api/browser/localstorage/set', () => ({
	__esModule: true,
	default: (key: string, value: string): boolean => setMock(key, value),
}));

describe('useHiddenServices', () => {
	beforeEach(() => store.clear());

	it('toggle hides then unhides, persisting to localStorage', () => {
		const { result } = renderHook(() => useHiddenServices());
		act(() => result.current.toggle('load-generator'));
		expect(result.current.hidden.has('load-generator')).toBe(true);
		expect(store.get(LOCALSTORAGE.NOC_TREND_HIDDEN)).toBe(
			JSON.stringify(['load-generator']),
		);
		act(() => result.current.toggle('load-generator'));
		expect(result.current.hidden.size).toBe(0);
		expect(store.get(LOCALSTORAGE.NOC_TREND_HIDDEN)).toBe(JSON.stringify([]));
	});

	it('restores hidden set from localStorage on mount', () => {
		store.set(LOCALSTORAGE.NOC_TREND_HIDDEN, JSON.stringify(['ad', 'frontend']));
		const { result } = renderHook(() => useHiddenServices());
		expect(result.current.hidden.has('ad')).toBe(true);
		expect(result.current.hidden.has('frontend')).toBe(true);
	});

	it('corrupt JSON falls back to empty set', () => {
		store.set(LOCALSTORAGE.NOC_TREND_HIDDEN, '{not json');
		const { result } = renderHook(() => useHiddenServices());
		expect(result.current.hidden.size).toBe(0);
	});
});
