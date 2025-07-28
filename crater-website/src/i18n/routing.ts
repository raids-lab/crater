import {defineRouting} from 'next-intl/routing';
import { locales, defaultLocale } from './config'; // [!code ++]

export const routing = defineRouting({
  locales: locales, // [!code ++]
  defaultLocale: defaultLocale, // [!code ++]
});