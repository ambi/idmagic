import { describe, expect, it } from 'bun:test'
import { render, screen } from '@testing-library/react'
import { LocaleProvider } from '../lib/i18n'
import { CharacterCount } from './CharacterCount'

function renderCount(value: string, max: number) {
  return render(
    <LocaleProvider initialLocale="en">
      <CharacterCount value={value} max={max} />
    </LocaleProvider>,
  )
}

describe('CharacterCount', () => {
  // 検証と同じくコードポイントで数える。絵文字は UTF-16 で 2 単位だが 1 文字である。
  it('shows the count in code points without announcing it far from the limit', () => {
    renderCount('😀😀', 10)
    expect(screen.getByText('2 / 10')).toBeInTheDocument()
    expect(screen.queryByText(/characters left/)).not.toBeInTheDocument()
  })

  it('announces the remaining characters near the limit', () => {
    renderCount('a'.repeat(9), 10)
    const announcement = screen.getByText('1 characters left.')
    expect(announcement.getAttribute('aria-live')).toBe('polite')
  })

  it('announces how far the value is over the limit', () => {
    renderCount('a'.repeat(12), 10)
    expect(screen.getByText('2 characters over the limit.')).toBeInTheDocument()
    expect(screen.getByText('12 / 10')).toBeInTheDocument()
  })
})
