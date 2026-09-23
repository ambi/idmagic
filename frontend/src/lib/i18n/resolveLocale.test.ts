import { describe, expect, it } from 'bun:test'

// 主要ユースケース追跡: REQ-SYSTEM-008、REQ-SYSTEM-010。
import { commonDictionary } from './common.i18n'
import { configuredDefaultLocale } from './locale'
import { resolveLocale } from './resolveLocale'

describe('resolveLocale', () => {
  //spec:covers EX-SYSTEM-008-01: 明示選択も保存済み設定も無いとき、認可リクエストの ui_locales ヒントが表示言語を決めること。
  it('uses the first supported ui_locales hint before a browser locale', () => {
    expect(resolveLocale({ uiLocalesHint: 'fr en ja', browserLanguages: ['ja-JP'] })).toBe('en')
  })

  // 保存済み設定は言語切り替え UI の明示選択だけから作られるので、RP の推定であるヒントより強い。
  it('prefers the saved explicit choice over a supported ui_locales hint', () => {
    expect(resolveLocale({ uiLocalesHint: 'en', saved: 'ja', browserLanguages: ['en-US'] })).toBe(
      'ja',
    )
  })

  it('uses the saved locale when no supported hint is present', () => {
    expect(resolveLocale({ uiLocalesHint: 'fr', saved: 'en', browserLanguages: ['ja-JP'] })).toBe(
      'en',
    )
  })

  // 未対応のブラウザー言語と対応するブラウザー言語を並べる。フォールバックの側だけを見る
  // 検査は、ブラウザー言語を一切読まない実装を通す。その実装では、対応する言語を設定して
  // いる利用者にも常に既定ロケールが出る。
  //
  //spec:covers EX-SYSTEM-004-01: 明示選択も保存済み設定も無く、ブラウザーの言語設定が未対応の fr であるとき、デフォルトロケールの en で表示すること。
  it('uses a supported browser language and otherwise falls back to the startup setting', () => {
    expect(resolveLocale({ browserLanguages: ['en-US'] })).toBe('en')
    expect(resolveLocale({ browserLanguages: ['fr-FR'] })).toBe('en')
    expect(resolveLocale({ browserLanguages: ['fr-FR'] }, 'ja')).toBe('ja')
  })

  // 起動時設定が使われる条件は、ほかのどの入力も無いことである。ja を与えた側だけを見る
  // 検査は、起動時設定を常に優先する実装を通す。
  //
  //spec:covers EX-SYSTEM-005-01: 明示選択・ui_locales ヒント・保存済み設定・対応するブラウザー言語がどれも無いとき、VITE_DEFAULT_LOCALE の ja で表示すること。
  it('uses the configured startup locale when nothing else resolves', () => {
    expect(resolveLocale({}, configuredDefaultLocale('ja'))).toBe('ja')
    expect(resolveLocale({ browserLanguages: ['en-US'] }, configuredDefaultLocale('ja'))).toBe('en')
  })

  //spec:covers EX-SYSTEM-005-02: VITE_DEFAULT_LOCALE が未設定または未対応値のとき、FallbackLocale の en で表示すること。
  it('uses English when the startup setting is absent or unsupported', () => {
    expect(configuredDefaultLocale()).toBe('en')
    expect(configuredDefaultLocale('fr')).toBe('en')
    expect(resolveLocale({}, configuredDefaultLocale('fr'))).toBe('en')
  })
})

describe('translation dictionaries', () => {
  it('has matching Japanese and English keys', () => {
    expect(Object.keys(commonDictionary.ja).sort()).toEqual(Object.keys(commonDictionary.en).sort())
  })
})
