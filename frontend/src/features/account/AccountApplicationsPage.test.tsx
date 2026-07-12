import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import {
  AccountApplicationsPage,
  AccountApplicationsPresentation,
  formatAccountConsentDate,
} from './AccountApplicationsPage'
import type { AccountConsent } from '../../types'

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

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

describe('AccountApplicationsPage', () => {
  const consent: AccountConsent = {
    client_id: 'client-1',
    client_name: 'Example App',
    scopes: ['openid', 'profile'],
    state: 'active',
    granted_at: '2026-01-01T00:00:00Z',
    expires_at: '2027-01-01T00:00:00Z',
  }

  afterEach(() => vi.unstubAllGlobals())

  it('revokes access and removes the application with a notice', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(204)))
    await renderWithRouter(
      <AccountApplicationsPage
        csrfToken="csrf"
        username="taro"
        isAdmin={false}
        consents={[consent]}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /アクセスを取り消す/ }))

    expect(await screen.findByText(/アクセスを取り消しました/)).toBeInTheDocument()
    expect(screen.queryByText('Example App')).not.toBeInTheDocument()
  })

  it('shows an error message when revoking fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(500, { message: '一時的に利用できません' })),
    )
    await renderWithRouter(
      <AccountApplicationsPage
        csrfToken="csrf"
        username="taro"
        isAdmin={false}
        consents={[consent]}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /アクセスを取り消す/ }))

    expect(await screen.findByText('一時的に利用できません')).toBeInTheDocument()
    expect(screen.getByText('Example App')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /アクセスを取り消す/ })).toBeEnabled(),
    )
  })
})
