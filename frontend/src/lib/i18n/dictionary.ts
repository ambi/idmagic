import type { Locale } from './locale'

const untranslatedKeysByDictionary = new WeakMap<object, string[]>()

// defineDictionary は ja を key 集合の正とし、en に同一 key 集合(値は string)を強制する。
// en 側の key 欠落・余剰は呼び出し箇所で型エラーになる (TranslationKeyIntegrity)。
// 型を迂回した値の欠落と空文字列は型検査で止まらないので、ja の該当 key は FallbackLocale
// (en) の値で埋める。フックを通らず辞書を直接引く箇所にも効くよう、定義の時点で解決する。
// en は落ちる先がないので、空文字列も訳として扱う (単位を付けない英語の接尾辞など)。
export function defineDictionary<T extends Record<string, string>>(
  ja: T,
  en: { [K in keyof T]: string },
): Record<Locale, T> {
  const fallback = en as T
  const resolved = { ...ja }
  const untranslated: string[] = []
  for (const key of Object.keys(fallback) as (keyof T & string)[]) {
    if (ja[key]) continue
    resolved[key] = fallback[key]
    untranslated.push(key)
  }
  const dictionary = { ja: resolved, en: fallback }
  untranslatedKeysByDictionary.set(dictionary, untranslated)
  return dictionary
}

// untranslatedKeys は defineDictionary が en の値で埋めた ja の key を返す。
// 実行時のフォールバックは翻訳漏れを画面から隠すので、全辞書を読む検査がこれで漏れを見つける。
export function untranslatedKeys(dictionary: Record<Locale, Record<string, string>>): string[] {
  return [...(untranslatedKeysByDictionary.get(dictionary) ?? [])]
}
