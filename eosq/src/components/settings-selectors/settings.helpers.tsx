import { i18n } from "../../i18n"

export const LANGUAGE_OPTIONS = [{ label: "EN", value: "en" }, { label: "中文", value: "zh" }]

export function getCurrentLanguageName() {
  const currentLang = getCurrentLanguageValue()

  const currentLanguage = LANGUAGE_OPTIONS.find((ref) => ref.value === currentLang)

  let languageName = "EN"
  if (currentLanguage) {
    languageName = currentLanguage.label
  }

  return languageName
}

export function getCurrentLanguageValue() {
  const usedLanguages = (i18n as any).languages

  const availableLanguages: string[] = LANGUAGE_OPTIONS.map(
    (language: { label: string; value: string }) => language.value
  )
  return usedLanguages.find((lang: string) => availableLanguages.includes(lang))
}
