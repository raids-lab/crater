// 定义支持的语言列表，包含语言代码和显示名称
export const locales = ['en', 'zh'] as const;

export type Locale = typeof locales[number];

export const localeNames: Record<typeof locales[number], string> = {
  en: 'English',
  zh: '简体中文',
};

// 如果需要，也可以定义一个默认语言
export const defaultLocale = 'zh';