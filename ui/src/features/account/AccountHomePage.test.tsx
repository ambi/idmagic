import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AccountHomePage, formatAccountSummaryDateTime } from './AccountHomePage'
import type { AccountSummary } from '../../types'

describe('formatAccountSummaryDateTime', () => {
  it('returns 記録なし for undefined values', async () => {
    expect(formatAccountSummaryDateTime(undefined)).toBe('記録なし')
  })

  it('returns 記録なし for invalid date strings', async () => {
    expect(formatAccountSummaryDateTime('not-a-date')).toBe('記録なし')
  })

  it('formats a valid ISO date string', async () => {
    const formatted = formatAccountSummaryDateTime('2026-01-15T10:30:00Z')
    expect(formatted).toContain('2026')
  })
})

describe('AccountHomePage', () => {
  const summary: AccountSummary = {
    sub: 'user-1',
    preferred_username: 'taro',
    name: 'Taro Yamada',
    email: 'taro@example.com',
    email_verified: true,
    mfa_enrolled: false,
    status: 'active',
    required_actions: [],
  }

  it('greets the user by display name', async () => {
    await renderWithRouter(<AccountHomePage summary={summary} isAdmin={false} />)
    expect(screen.getByText('こんにちは、Taro Yamada さん')).toBeInTheDocument()
  })

  it('shows required actions when present', async () => {
    await renderWithRouter(
      <AccountHomePage
        summary={{ ...summary, required_actions: ['verify_email'] }}
        isAdmin={false}
      />,
    )
    expect(screen.getByText('対応が必要な項目があります')).toBeInTheDocument()
  })

  it('does not show required actions section when empty', async () => {
    await renderWithRouter(<AccountHomePage summary={summary} isAdmin={false} />)
    expect(screen.queryByText('対応が必要な項目があります')).not.toBeInTheDocument()
  })

  it('shows MFA as unregistered when not enrolled', async () => {
    await renderWithRouter(<AccountHomePage summary={summary} isAdmin={false} />)
    expect(screen.getByText('未登録')).toBeInTheDocument()
  })
})
