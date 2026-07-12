import type { Locale } from './locale'

// defineDictionary は ja を key 集合の正とし、en に同一 key 集合(値は string)を強制する。
// en 側の key 欠落・余剰は呼び出し箇所で型エラーになる (TranslationKeyIntegrity)。
export function defineDictionary<T extends Record<string, string>>(
  ja: T,
  en: { [K in keyof T]: string },
): Record<Locale, T> {
  return { ja, en: en as T }
}
