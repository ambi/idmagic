import { FALLBACK_LOCALE, type Locale, parseLocaleTag } from './locale'

export interface LocaleResolutionInput {
  // OIDC ui_locales hint (space区切りの候補列, RFC の並び順)。
  uiLocalesHint?: string | null
  // 保存済み設定 (localStorage 由来)。
  saved?: string | null
  // ブラウザ言語 (navigator.languages)。
  browserLanguages?: readonly string[]
}

// resolveLocale は UX-LOCALE の解決順 (明示選択 > ui_locales hint > 保存済み設定 >
// ブラウザ言語 > 既定locale) のうち、起動時に評価する後半4段を1回だけ評価する。
// 「明示選択」は言語切り替え UI が直接 state を更新して以後の再解決を起こさないことで
// 表現され、ui_locales hint を含む後続の解決より常に優先される。
export function resolveLocale(input: LocaleResolutionInput): Locale {
  const hintCandidates = input.uiLocalesHint?.split(/\s+/).filter(Boolean) ?? []
  const candidates: (string | null | undefined)[] = [
    ...hintCandidates,
    input.saved,
    ...(input.browserLanguages ?? []),
  ]
  for (const candidate of candidates) {
    const parsed = parseLocaleTag(candidate)
    if (parsed) return parsed
  }
  return FALLBACK_LOCALE
}
