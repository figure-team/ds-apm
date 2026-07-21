import i18n from 'i18next';

// Translation for non-hook code (utils, mutation callbacks). Uses the shared
// i18next singleton initialized at app bootstrap (src/index.tsx -> ReactI18).
// Falls back to the key itself when the singleton is not initialized (jest) —
// 초기화 전 t() 호출이 throw하면 렌더 경로 전체가 죽으므로 반드시 삼킨다.
export const i18nText = (key: string): string => {
	try {
		const translated = i18n.t(key);
		return typeof translated === 'string' ? translated : key;
	} catch {
		return key;
	}
};
