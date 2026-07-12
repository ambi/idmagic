import { FALLBACK_LOCALE, type Locale } from './locale'

// api/core.ts のような非 React モジュールが、現在の DisplayLanguage を同期的に
// 参照するための最小限の橋渡し (bearerTokenProvider と同じパターン)。
// LocaleProvider がロケール変更のたびに setCurrentLocale を呼ぶ。
let current: Locale = FALLBACK_LOCALE

export function getCurrentLocale(): Locale {
  return current
}

export function setCurrentLocale(locale: Locale): void {
  current = locale
}
