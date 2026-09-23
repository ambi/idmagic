import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import { defineDictionary, LocaleProvider, useDictionary, useFormatters } from './index'
import { commonDictionary } from './common.i18n'

// 主要ユースケース追跡: REQ-SYSTEM-010。

// Probe は画面が i18n から受け取るものを 1 か所に並べる。文言と書式の 2 つが同じ選択に
// 従うことを、同じ木の中で読むためである。
function Probe() {
  const t = useDictionary(commonDictionary)
  const { formatDate, formatDateTime, formatNumber } = useFormatters()
  return (
    <div>
      <span data-testid="label">{t.languageSwitcherLabel}</span>
      <span data-testid="date">{formatDate('2026-09-13T04:05:06Z')}</span>
      <span data-testid="datetime">{formatDateTime('2026-09-13T04:05:06Z')}</span>
      <span data-testid="number">{formatNumber(1234567.5)}</span>
    </div>
  )
}

describe('the selected locale reaches both the dictionary and the formatters', () => {
  // 文言だけを見る検査は、日時と数値を locale と無関係に組み立てる実装を通す。それが
  // いちばん起きやすい退行であり (`String(value)` や固定の 'en')、辞書の側からは直せない。
  // 書式の判定に固定の文字列を使わないのは、ICU のバージョンで表記が動くためである。en と ja を
  // 区別する印 (月の略称と年から始まる並び) と、両者が異なることだけを見る。
  //
  //spec:covers EX-SYSTEM-010-01: en を選んだ画面が en 辞書の文言と en の日時・数値の書式で表示すること。
  //spec:covers EX-SYSTEM-010-02: ja を選ぶと同じ要素が ja 辞書と ja の書式で表示されること。
  it('renders dictionary text and date and number formats per locale', () => {
    const english = render(
      <LocaleProvider initialLocale="en">
        <Probe />
      </LocaleProvider>,
    )
    const en = {
      label: screen.getByTestId('label').textContent,
      date: screen.getByTestId('date').textContent ?? '',
      datetime: screen.getByTestId('datetime').textContent ?? '',
      number: screen.getByTestId('number').textContent,
    }
    expect(en.label).toBe(commonDictionary.en.languageSwitcherLabel)
    expect(en.date).toContain('Sep')
    expect(en.datetime).toContain('Sep')
    expect(en.number).toBe('1,234,567.5')
    english.unmount()

    render(
      <LocaleProvider initialLocale="ja">
        <Probe />
      </LocaleProvider>,
    )
    const ja = {
      label: screen.getByTestId('label').textContent,
      date: screen.getByTestId('date').textContent ?? '',
      datetime: screen.getByTestId('datetime').textContent ?? '',
      number: screen.getByTestId('number').textContent,
    }
    expect(ja.label).toBe(commonDictionary.ja.languageSwitcherLabel)
    expect(ja.date).toContain('2026/')
    expect(ja.datetime).toContain('2026/')
    expect(ja.number).toBe('1,234,567.5')

    expect(ja.date).not.toBe(en.date)
    expect(ja.datetime).not.toBe(en.datetime)
    expect(ja.label).not.toBe(en.label)
  })

  // Provider を持たない presentation の単体テストが製品と同じ既定を見ることの確認である。
  // 具体例はどれもこの木を述べていないので、id は名指さない。
  it('falls back to English without a provider', () => {
    render(<Probe />)
    expect(screen.getByTestId('label').textContent).toBe(commonDictionary.en.languageSwitcherLabel)
    expect(screen.getByTestId('date').textContent).toContain('Sep')
  })
})

describe('a key the selected dictionary lacks', () => {
  // 空文字列は型検査を通る。値を持たないキーは、型を `as` で迂回した辞書を表す。
  const partial = defineDictionary(
    { translated: '訳あり', empty: '', missing: undefined as unknown as string },
    { translated: 'Translated', empty: 'Empty in ja', missing: 'Missing in ja' },
  )

  function PartialProbe() {
    const t = useDictionary(partial)
    return (
      <div>
        <span data-testid="translated">{t.translated}</span>
        <span data-testid="empty">{t.empty}</span>
        <span data-testid="missing">{t.missing}</span>
      </div>
    )
  }

  //spec:covers EX-SYSTEM-010-03: ja を選んだ画面で ja 辞書に訳が無いキーは、en 辞書の同じキーの文言で表示されること。
  it('shows the en text for a key the ja dictionary lacks', () => {
    render(
      <LocaleProvider initialLocale="ja">
        <PartialProbe />
      </LocaleProvider>,
    )
    expect(screen.getByTestId('translated').textContent).toBe('訳あり')
    expect(screen.getByTestId('empty').textContent).toBe('Empty in ja')
    expect(screen.getByTestId('missing').textContent).toBe('Missing in ja')
  })
})
