// UX-LOCALE (spec/contexts/system.yaml): idmagic は ja/en のみをサポート対象とする。
export const SUPPORTED_LOCALES = ['ja', 'en'] as const

export type Locale = (typeof SUPPORTED_LOCALES)[number]

// FallbackLocale (glossary): 未対応 locale・辞書 key 欠落時に用いる既定 locale。
export const FALLBACK_LOCALE: Locale = 'ja'

export function isSupportedLocale(value: string | null | undefined): value is Locale {
  return value != null && (SUPPORTED_LOCALES as readonly string[]).includes(value)
}

// parseLocaleTag は BCP47 言語タグ ("en-US" など) から対応 primary language を取り出す。
export function parseLocaleTag(tag: string | null | undefined): Locale | undefined {
  if (!tag) return undefined
  const primary = tag.split('-')[0]?.toLowerCase()
  return isSupportedLocale(primary) ? primary : undefined
}
