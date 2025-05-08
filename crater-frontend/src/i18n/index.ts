import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

import enTranslation from "./locales/en/translation.json";
import zhTranslation from "./locales/zh/translation.json";
import jpTranslation from "./locales/jp/translation.json";
import koTranslation from "./locales/ko/translation.json";

i18n
  .use(LanguageDetector)
  .use(initReactI18next) // passes i18n down to react-i18next
  .init({
    resources: {
      en: {
        translation: enTranslation,
      },
      zh: {
        translation: zhTranslation,
      },
      jp: {
        translation: jpTranslation,
      },
      ko: {
        translation: koTranslation,
      },
    },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage", "sessionStorage", "cookie"],
    },
    fallbackLng: "zh", // language to use, more information here: https://www.i18next.com/overview/configuration-options#languages-namespaces-resources
    // you can use the i18n.changeLanguage function to change the language manually: https://www.i18next.com/overview/api#changelanguage
    // if you're using a language detector, do not define the lng option
    interpolation: {
      escapeValue: false, // react already safes from xss
    },
  });

export default i18n;
