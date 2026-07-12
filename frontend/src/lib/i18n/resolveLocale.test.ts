import { describe, expect, it } from 'vitest'
import { commonDictionary } from './common.i18n'
import { configuredDefaultLocale } from './locale'
import { resolveLocale } from './resolveLocale'

describe('resolveLocale', () => {
  it('uses the first supported ui_locales hint before a saved or browser locale', () => {
    expect(
      resolveLocale({
        uiLocalesHint: 'fr en ja',
        saved: 'ja',
        browserLanguages: ['ja-JP'],
      }),
    ).toBe('en')
  })

  it('uses the saved locale when no supported hint is present', () => {
    expect(resolveLocale({ uiLocalesHint: 'fr', saved: 'en', browserLanguages: ['ja-JP'] })).toBe(
      'en',
    )
  })

  it('uses a supported browser language and otherwise falls back to the startup setting', () => {
    expect(resolveLocale({ browserLanguages: ['en-US'] })).toBe('en')
    expect(resolveLocale({ browserLanguages: ['fr-FR'] }, 'ja')).toBe('ja')
  })

  it('uses English when the startup setting is absent or unsupported', () => {
    expect(configuredDefaultLocale()).toBe('en')
    expect(configuredDefaultLocale('fr')).toBe('en')
  })
})

describe('translation dictionaries', () => {
  it('has matching Japanese and English keys', () => {
    expect(Object.keys(commonDictionary.ja).sort()).toEqual(Object.keys(commonDictionary.en).sort())
  })
})
