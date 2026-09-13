import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'bun:test'
import { LocaleProvider } from '../lib/i18n'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { LanguageSwitcher } from './LanguageSwitcher'

describe('LanguageSwitcher', () => {
  afterEach(() => {
    window.localStorage.clear()
    document.documentElement.lang = ''
  })

  it('uses English by default and persists an explicit Japanese choice', () => {
    render(
      <LocaleProvider>
        <LanguageSwitcher />
      </LocaleProvider>,
    )

    expect(screen.getByRole('button', { name: 'English' })).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(screen.getByRole('button', { name: '日本語' }))

    expect(screen.getByRole('button', { name: '日本語' })).toHaveAttribute('aria-pressed', 'true')
    expect(document.documentElement.lang).toBe('ja')
    expect(window.localStorage.getItem('idmagic.displayLocale')).toBe('ja')
  })

  // 明示選択の効き方を 3 点で見る。文言が選んだ辞書へ替わること、選択がブラウザーへ
  // 保存されること、そして次の表示でその保存が優先されることである。保存だけを見る検査は、
  // 保存はするが読み直さない実装を通す。その実装では、利用者が毎回選び直すことになる。
  //
  //spec:covers EX-SYSTEM-003-01: 表示言語に en を明示選択すると文言が en 辞書へ替わり、選択がブラウザーへ保存されて以後の表示で優先されること。
  it('switches the dictionary on an explicit English choice and prefers it next time', () => {
    const first = render(
      <LocaleProvider initialLocale="ja">
        <LanguageSwitcher />
      </LocaleProvider>,
    )
    expect(screen.getByRole('group')).toHaveAttribute(
      'aria-label',
      commonDictionary.ja.languageSwitcherLabel,
    )

    fireEvent.click(screen.getByRole('button', { name: 'English' }))

    expect(screen.getByRole('group')).toHaveAttribute(
      'aria-label',
      commonDictionary.en.languageSwitcherLabel,
    )
    expect(document.documentElement.lang).toBe('en')
    expect(window.localStorage.getItem('idmagic.displayLocale')).toBe('en')
    first.unmount()

    // 次のアクセス。ブラウザー言語は ja だが、保存済みの明示選択が優先される。
    render(
      <LocaleProvider>
        <LanguageSwitcher />
      </LocaleProvider>,
    )
    expect(screen.getByRole('button', { name: 'English' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('group')).toHaveAttribute(
      'aria-label',
      commonDictionary.en.languageSwitcherLabel,
    )
  })
})
