import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen, within } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { IdentityProviderConnection } from '../../api'
import { AdminIdentityProvidersPage } from './AdminIdentityProvidersPage'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'

const t = identityProvidersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

function connection(
  overrides: Partial<IdentityProviderConnection> = {},
): IdentityProviderConnection {
  return {
    id: 'google',
    tenant_id: 'tenant-1',
    display_name: 'Google',
    protocol: 'oidc',
    status: 'disabled',
    issuer: 'https://accounts.example.com',
    client_id: 'client-1',
    client_secret_configured: true,
    authorization_endpoint: 'https://accounts.example.com/authorize',
    token_endpoint: 'https://accounts.example.com/token',
    jwks_uri: 'https://accounts.example.com/jwks',
    claim_mapping: { subject: 'sub', username: 'email' },
    linking_policy: 'none',
    jit_provisioning: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('AdminIdentityProvidersPage', () => {
  afterEach(() => restoreGlobals())

  it('renders the connection list', async () => {
    await renderWithRouter(
      <AdminIdentityProvidersPage csrfToken="csrf" connections={[connection()]} />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText('Google')).toBeInTheDocument()
  })

  // RED (Design: 状態モデルの単純化): Disabled (旧 Draft 相当) の接続は確認なしで
  // 即座に削除できる。
  it('deletes a disabled connection immediately without a confirmation dialog', async () => {
    const fetchMock = mock((url: string, init?: RequestInit) => {
      if (init?.method === 'DELETE') return Promise.resolve(response(204))
      return Promise.resolve(response(200, {}))
    })
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(
      <AdminIdentityProvidersPage csrfToken="csrf" connections={[connection()]} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await screen.findByText(t.empty)
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/admin/v1/identity-providers/google'),
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  // RED: Active な接続の削除はフロントエンドで確認ダイアログを要求し、確認するまで
  // DELETE リクエストは送らない (Design §状態モデルの単純化)。
  it('requires confirmation before deleting an active connection', async () => {
    const fetchMock = mock((_url: string, init?: RequestInit) => {
      if (init?.method === 'DELETE') return Promise.resolve(response(204))
      return Promise.resolve(response(200, {}))
    })
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(
      <AdminIdentityProvidersPage
        csrfToken="csrf"
        connections={[connection({ status: 'active' })]}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    const dialog = await screen.findByRole('dialog')
    expect(fetchMock).not.toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ method: 'DELETE' }),
    )
    fireEvent.click(within(dialog).getByRole('button', { name: t.deleteConfirmButton }))
    await screen.findByText(t.empty)
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/admin/v1/identity-providers/google'),
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  // RED (interface: TestIdentityProviderConnection): a failing test result is
  // shown as a clear banner listing failure reasons, not a generic success toast.
  it('shows a failure banner with reasons when the connection test fails', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/test')) {
          return Promise.resolve(
            response(200, { result: { success: false, failures: ['JWKS URI is unreachable'] } }),
          )
        }
        return Promise.resolve(response(200, {}))
      }),
    )
    await renderWithRouter(
      <AdminIdentityProvidersPage csrfToken="csrf" connections={[connection()]} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.test }))
    await screen.findByText(t.testResultFailureHeading)
    expect(screen.getByText('JWKS URI is unreachable')).toBeInTheDocument()
  })
})
