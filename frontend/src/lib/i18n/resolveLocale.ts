import { configuredDefaultLocale, type Locale, parseLocaleTag } from './locale'

export interface LocaleResolutionInput {
  // OIDC ui_locales hint (space区切りの候補列, RFC の並び順)。
  uiLocalesHint?: string | null
  // 保存済み設定 (localStorage 由来)。言語切り替え UI の明示選択だけが書き込む。
  saved?: string | null
  // ブラウザ言語 (navigator.languages)。
  browserLanguages?: readonly string[]
}

// resolveLocale は起動時の表示言語を、保存済みの明示選択 > ui_locales hint > ブラウザ言語 >
// 既定locale の順に1回だけ評価する。ui_locales は OP が従わなくてもよい RP 側の推定であり、
// 利用者が OP の画面で自分で選んだ言語より弱い。同じページの寿命の中の明示選択は、
// 言語切り替え UI が直接 state を更新して再解決を起こさないことで表現される。
export function resolveLocale(
  input: LocaleResolutionInput,
  startupDefault = configuredDefaultLocale(),
): Locale {
  const hintCandidates = input.uiLocalesHint?.split(/\s+/).filter(Boolean) ?? []
  const candidates: (string | null | undefined)[] = [
    input.saved,
    ...hintCandidates,
    ...(input.browserLanguages ?? []),
  ]
  for (const candidate of candidates) {
    const parsed = parseLocaleTag(candidate)
    if (parsed) return parsed
  }
  return startupDefault
}
