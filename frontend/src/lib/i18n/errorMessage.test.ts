import { describe, expect, it } from 'bun:test'
import { commonDictionary } from './common.i18n'
import { localizedErrorMessage } from './errorMessage'

describe('localizedErrorMessage', () => {
  //spec:covers EX-SYSTEM-011-01: バックエンドが既知の stable エラーコードを返したとき、選択済みの表示言語の辞書にあるエラー文を表示すること。
  it('maps a known code to the localized dictionary value', () => {
    expect(localizedErrorMessage('en', 'access_denied', 'fallback')).toBe(
      commonDictionary.en.accessDenied,
    )
    expect(localizedErrorMessage('ja', 'access_denied', 'fallback')).toBe(
      commonDictionary.ja.accessDenied,
    )
  })

  // 未知のコードでバックエンドの文をそのまま出すことが、翻訳の範囲を stable なコードに
  // 限る約束の裏側である。勝手に訳す実装も、辞書に無いコードで空文字を出す実装も、
  // ここで落ちる。
  //
  //spec:covers EX-SYSTEM-011-02: エラーコードが未知のとき、バックエンドが返した人間可読文をそのまま表示すること。
  it('returns the fallback for an unknown code', () => {
    expect(localizedErrorMessage('en', 'unknown_code', 'fallback text')).toBe('fallback text')
    expect(localizedErrorMessage('en', undefined, 'fallback text')).toBe('fallback text')
  })

  it('maps rate_limited to the localized dictionary value without a retry-after suffix', () => {
    expect(localizedErrorMessage('en', 'rate_limited', 'fallback')).toBe(
      commonDictionary.en.rateLimited,
    )
    expect(localizedErrorMessage('ja', 'rate_limited', 'fallback')).toBe(
      commonDictionary.ja.rateLimited,
    )
  })

  it('appends a localized retry-after sentence when retryAfterSeconds is given', () => {
    const en = localizedErrorMessage('en', 'rate_limited', 'fallback', 30)
    expect(en.startsWith(commonDictionary.en.rateLimited)).toBe(true)
    expect(en).toContain('30')

    const ja = localizedErrorMessage('ja', 'rate_limited', 'fallback', 30)
    expect(ja.startsWith(commonDictionary.ja.rateLimited)).toBe(true)
    expect(ja).toContain('30')
  })

  it('does not append a retry-after sentence for other error codes', () => {
    const message = localizedErrorMessage('en', 'access_denied', 'fallback', 30)
    expect(message).toBe(commonDictionary.en.accessDenied)
    expect(message).not.toContain('30')
  })
})
