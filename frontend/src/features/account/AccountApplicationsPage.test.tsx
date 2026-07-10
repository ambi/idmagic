import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import {
  AccountApplicationsPresentation,
  formatAccountConsentDate,
} from './AccountApplicationsPage'
import type { AccountConsent } from '../../types'

describe('formatAccountConsentDate', () => {
  it('formats a valid ISO date string', () => {
    expect(formatAccountConsentDate('2026-01-15T10:30:00Z')).toContain('2026')
  })

  it('returns the raw value for an invalid date string', () => {
    expect(formatAccountConsentDate('not-a-date')).toBe('not-a-date')
  })
})

describe('AccountApplicationsPresentation', () => {
  const consent: AccountConsent = {
    client_id: 'client-1',
    client_name: 'Example App',
    scopes: ['openid', 'profile'],
    state: 'active',
    granted_at: '2026-01-01T00:00:00Z',
    expires_at: '2027-01-01T00:00:00Z',
  }

  const baseProps = {
    username: 'taro',
    isAdmin: false,
    consents: [consent],
    pending: '',
    error: '',
    notice: '',
    onDismissNotice: vi.fn(),
    onRevoke: vi.fn(),
  }

  it('shows an empty state when there are no consents', async () => {
    await renderWithRouter(<AccountApplicationsPresentation {...baseProps} consents={[]} />)
    expect(screen.getByText('アクセスを許可したアプリはありません。')).toBeInTheDocument()
  })

  it('renders consent details', async () => {
    await renderWithRouter(<AccountApplicationsPresentation {...baseProps} />)
    expect(screen.getByText('Example App')).toBeInTheDocument()
    expect(screen.getByText('openid')).toBeInTheDocument()
  })

  it('calls onRevoke with the consent when revoke is clicked', async () => {
    const onRevoke = vi.fn()
    await renderWithRouter(<AccountApplicationsPresentation {...baseProps} onRevoke={onRevoke} />)
    fireEvent.click(screen.getByRole('button', { name: /アクセスを取り消す/ }))
    expect(onRevoke).toHaveBeenCalledWith(consent)
  })

  it('disables the revoke button while pending for that client', async () => {
    await renderWithRouter(<AccountApplicationsPresentation {...baseProps} pending="client-1" />)
    expect(screen.getByRole('button', { name: /取り消し中/ })).toBeDisabled()
  })

  it('shows an error message when present', async () => {
    await renderWithRouter(<AccountApplicationsPresentation {...baseProps} error="失敗しました" />)
    expect(screen.getByText('失敗しました')).toBeInTheDocument()
  })
})
