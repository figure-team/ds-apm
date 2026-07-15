import i18n from 'i18next';

// Translation for non-hook code (utils, mutation callbacks). Uses the shared
// i18next singleton initialized at app bootstrap (src/index.tsx -> ReactI18).
// Falls back to the key itself when the singleton is not initialized (jest).
export const i18nText = (key: string): string => {
	const translated = i18n.t(key);
	return typeof translated === 'string' ? translated : key;
};
