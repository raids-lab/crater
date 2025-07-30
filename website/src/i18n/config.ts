// 定义支持的语言列表，包含语言代码和显示名称

export const supportedLocales = {
  en: 'English',
  zh: '简体中文',
  jp: '日本語'
} as const;

export type Locale = keyof typeof supportedLocales;

export const locales = Object.keys(supportedLocales) as Locale[];

export const localeNames: Record<Locale, string> = supportedLocales;

export const defaultLocale: Locale = 'zh';