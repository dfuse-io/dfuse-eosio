import i18nLib from "i18next"
import LanguageDetector from "i18next-browser-languagedetector"
import { reactI18nextModule } from "react-i18next"
import { en } from "./en"
import { zh } from "./zh"

i18nLib
  .use(LanguageDetector)
  .use(reactI18nextModule)
  .init({
    fallbackLng: "en",
    ns: ["translations"],
    defaultNS: "translations",
    lookupCookie: "i18next",
    resources: {
      en: {
        translations: en
      },
      zh: {
        translations: zh
      }
    },

    debug: true,

    interpolation: {
      // React already escape stuff correctly, so not needed
      escapeValue: false
    },

    react: {
      wait: true
    }
  } as any)

export const i18n = i18nLib
export const t = i18n.t.bind(i18n)
