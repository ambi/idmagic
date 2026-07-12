import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { LocaleProvider } from '../lib/i18n'
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
})
