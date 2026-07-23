import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ApiTokensTab } from './ApiTokensTab'

const t = adminSettingsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

describe('ApiTokensTab', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('selects scopes, issues a JWT once, lists scopes, and revokes it', async () => {
    const meta = {
      id: 'token-1',
      description: 'Okta',
      scopes: ['scim:users:read', 'scim:users:write'],
      created_at: '2026-07-23T00:00:00Z',
    }
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(response(200, { tokens: [] }))
        .mockResolvedValueOnce(response(201, { token: 'header.payload.signature', meta }))
        .mockResolvedValueOnce(response(200, { tokens: [meta] }))
        .mockResolvedValueOnce(response(204)),
    )

    await renderWithRouter(<ApiTokensTab csrfToken="csrf" tenantRealm="default" />)
    await screen.findByText(t.noTokensNotice)
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
    fireEvent.change(screen.getByLabelText(t.tokenDescriptionLabel), { target: { value: 'Okta' } })
    fireEvent.click(screen.getByLabelText('scim:users:read'))
    fireEvent.click(screen.getByLabelText('scim:users:write'))
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    expect(await screen.findByDisplayValue('header.payload.signature')).toBeInTheDocument()
    expect(await screen.findByText('scim:users:read')).toBeInTheDocument()
    const post = vi.mocked(fetch).mock.calls.find(([, init]) => init?.method === 'POST')
    expect(post?.[1]?.body).toBe(
      JSON.stringify({
        description: 'Okta',
        scopes: ['scim:users:read', 'scim:users:write'],
        expiry_days: 7,
      }),
    )

    fireEvent.click(screen.getByRole('button', { name: t.revoke }))
    await waitFor(() => expect(screen.queryByText('scim:users:read')).not.toBeInTheDocument())
  })

  it('offers account self-service scopes without a client selection', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { tokens: [] })))
    await renderWithRouter(<ApiTokensTab csrfToken="csrf" tenantRealm="default" />)
    await screen.findByText(t.noTokensNotice)
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
    for (const scope of [
      'account:read',
      'account:write',
      'account:mfa:write',
      'account:sessions:write',
      'account:consents:write',
      'account:password:write',
    ]) {
      expect(screen.getByLabelText(scope)).toBeInTheDocument()
    }
    expect(screen.getByLabelText('account:mfa:write').closest('label')).toHaveClass(
      'sm:col-start-2',
    )
    expect(screen.getByLabelText('account:write').closest('label')).not.toHaveClass(
      'sm:col-start-2',
    )
    expect(screen.queryByRole('combobox', { name: /client/i })).not.toBeInTheDocument()
  })

  it('offers application and protocol management scopes', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { tokens: [] })))

    await renderWithRouter(<ApiTokensTab csrfToken="csrf" tenantRealm="default" />)
    await screen.findByText(t.noTokensNotice)
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    for (const scope of [
      'applications:read',
      'applications:write',
      'oauth-clients:read',
      'oauth-clients:write',
      'authorization-detail-types:read',
      'authorization-detail-types:write',
      'mcp-resource-servers:read',
      'mcp-resource-servers:write',
      'saml:read',
      'saml:write',
      'wsfed:read',
      'wsfed:write',
      'provisioning:read',
      'provisioning:write',
    ]) {
      expect(screen.getByLabelText(scope)).toBeInTheDocument()
    }
  })

  it('shows all API base URLs and groups scopes with human-readable guidance', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { tokens: [] })))

    await renderWithRouter(<ApiTokensTab csrfToken="csrf" tenantRealm="acme" />)
    await screen.findByText(t.noTokensNotice)

    expect(screen.getByLabelText(t.managementApiBaseUrlLabel)).toHaveValue(
      'http://localhost:3000/realms/acme/api/admin',
    )
    expect(screen.getByLabelText(t.scimBaseUrlLabel)).toHaveValue(
      'http://localhost:3000/realms/acme/scim/v2',
    )
    expect(screen.getByLabelText(t.accountApiBaseUrlLabel)).toHaveValue(
      'http://localhost:3000/realms/acme/api/account',
    )

    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
    expect(screen.getByText(t.managementScopesHeading)).toBeInTheDocument()
    expect(screen.getByText(t.scimScopesHeading)).toBeInTheDocument()
    expect(screen.getByText(t.accountScopesHeading)).toBeInTheDocument()
    expect(screen.getByText(t.managementScopesHeading).closest('details')).not.toHaveAttribute(
      'open',
    )
    expect(screen.getAllByText(t.usersScopeResourceLabel).length).toBeGreaterThan(0)
    expect(screen.getByText(t.readScopeDescription)).toBeInTheDocument()
  })
})
