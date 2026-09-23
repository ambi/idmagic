import { describe, expect, it } from 'bun:test'
import { defineDictionary, untranslatedKeys } from './dictionary'

// 主要ユースケース追跡: REQ-SYSTEM-010。

describe('defineDictionary', () => {
  it('fills a missing or empty ja entry from the en entry', () => {
    const dictionary = defineDictionary(
      { translated: '訳あり', empty: '', missing: undefined as unknown as string },
      { translated: 'Translated', empty: 'Empty in ja', missing: 'Missing in ja' },
    )

    expect(dictionary.ja).toEqual({
      translated: '訳あり',
      empty: 'Empty in ja',
      missing: 'Missing in ja',
    })
  })

  // en は FallbackLocale そのものなので落ちる先がない。空文字列も訳としてそのまま残す。
  it('keeps an empty en entry as its translation', () => {
    const dictionary = defineDictionary({ countSuffix: ' 件' }, { countSuffix: '' })

    expect(dictionary.en.countSuffix).toBe('')
    expect(untranslatedKeys(dictionary)).toEqual([])
  })

  it('reports the ja entries it had to fill', () => {
    const dictionary = defineDictionary(
      { translated: '訳あり', empty: '', missing: undefined as unknown as string },
      { translated: 'Translated', empty: 'Empty in ja', missing: 'Missing in ja' },
    )

    expect(untranslatedKeys(dictionary)).toEqual(['empty', 'missing'])
  })
})
