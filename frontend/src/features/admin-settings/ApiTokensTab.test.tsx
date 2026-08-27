import { act, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ApiTokensTab } from './ApiTokensTab'

const t = adminSettingsDictionary.en

const integrationEndpoints = {
  issuer: 'https://login.idmagic.example/realms/acme',
  oauth: {
    openid_configuration:
      'https://login.idmagic.example/realms/acme/.well-known/openid-configuration',
    oauth_authorization_server:
      'https://login.idmagic.example/realms/acme/.well-known/oauth-authorization-server',
    authorization_endpoint: 'https://login.idmagic.example/realms/acme/authorize',
    token_endpoint: 'https://login.idmagic.example/realms/acme/token',
    userinfo_endpoint: 'https://login.idmagic.example/realms/acme/userinfo',
    jwks_uri: 'https://login.idmagic.example/realms/acme/jwks',
    revocation_endpoint: 'https://login.idmagic.example/realms/acme/revoke',
    introspection_endpoint: 'https://login.idmagic.example/realms/acme/introspect',
    end_session_endpoint: 'https://login.idmagic.example/realms/acme/logout',
    registration_endpoint: 'https://login.idmagic.example/realms/acme/register',
    pushed_authorization_request_endpoint: 'https://login.idmagic.example/realms/acme/par',
    device_authorization_endpoint: 'https://login.idmagic.example/realms/acme/device_authorization',
  },
  saml: {
    entity_id: 'https://login.idmagic.example/realms/acme/saml',
    metadata_url: 'https://login.idmagic.example/realms/acme/saml/metadata',
    sso_url: 'https://login.idmagic.example/realms/acme/saml/sso',
    slo_url: 'https://login.idmagic.example/realms/acme/saml/slo',
    signing_certificate: {
      download_url: 'https://login.idmagic.example/realms/acme/saml/signing-certificate.pem',
      fingerprint_sha256: 'AA:BB',
      not_before: '2026-01-01T00:00:00Z',
      not_after: '2027-01-01T00:00:00Z',
    },
  },
  ws_federation: {
    realm: 'https://login.idmagic.example/realms/acme',
    metadata_url:
      'https://login.idmagic.example/realms/acme/federationmetadata/2007-06/federationmetadata.xml',
    passive_logon_url: 'https://login.idmagic.example/realms/acme/wsfed',
    active_logon_url: 'https://login.idmagic.example/realms/acme/trust/usernamemixed',
    metadata_exchange_url: 'https://login.idmagic.example/realms/acme/trust/mex',
  },
  apis: {
    management_api_base_url: 'https://api.idmagic.example/management/acme',
    scim_base_url: 'https://api.idmagic.example/scim/acme/v2',
    account_api_base_url: 'https://api.idmagic.example/account/acme',
  },
} as const

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('ApiTokensTab', () => {
  afterEach(() => restoreGlobals())

  it('selects scopes, issues a JWT once, lists scopes, and revokes it', async () => {
    const meta = {
      id: 'token-1',
      description: 'Okta',
      scopes: ['scim:users:read', 'scim:users:write'],
      created_at: '2026-07-23T00:00:00Z',
    }
    const fetchMock = mock()
      .mockResolvedValueOnce(response(200, { tokens: [] }))
      .mockResolvedValueOnce(response(201, { token: 'header.payload.signature', meta }))
      .mockResolvedValueOnce(response(200, { tokens: [meta] }))
      .mockResolvedValueOnce(response(204))
    stubGlobal('fetch', fetchMock)

    await renderWithRouter(
      <ApiTokensTab csrfToken="csrf" integrationEndpoints={integrationEndpoints} />,
    )
    await screen.findByText(t.noTokensNotice)
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
    fireEvent.change(screen.getByLabelText(t.tokenDescriptionLabel), { target: { value: 'Okta' } })
    fireEvent.click(screen.getByLabelText('scim:users:read'))
    fireEvent.click(screen.getByLabelText('scim:users:write'))
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    expect(await screen.findByDisplayValue('header.payload.signature')).toBeInTheDocument()
    expect(await screen.findByText('scim:users:read')).toBeInTheDocument()
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')
    expect(post?.[1]?.body).toBe(
      JSON.stringify({
        description: 'Okta',
        scopes: ['scim:users:read', 'scim:users:write'],
        expiry_days: 7,
      }),
    )

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: t.revoke }))
      await new Promise((resolve) => setTimeout(resolve, 0))
    })
    expect(screen.queryByText('scim:users:read')).not.toBeInTheDocument()
  })

  it('offers account self-service scopes without a client selection', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { tokens: [] })))
    await renderWithRouter(
      <ApiTokensTab csrfToken="csrf" integrationEndpoints={integrationEndpoints} />,
    )
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
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { tokens: [] })))

    await renderWithRouter(
      <ApiTokensTab csrfToken="csrf" integrationEndpoints={integrationEndpoints} />,
    )
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
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { tokens: [] })))

    await renderWithRouter(
      <ApiTokensTab csrfToken="csrf" integrationEndpoints={integrationEndpoints} />,
    )
    await screen.findByText(t.noTokensNotice)

    expect(screen.getByLabelText(t.managementApiBaseUrlLabel)).toHaveValue(
      'https://api.idmagic.example/management/acme',
    )
    expect(screen.getByLabelText(t.scimBaseUrlLabel)).toHaveValue(
      'https://api.idmagic.example/scim/acme/v2',
    )
    expect(screen.getByLabelText(t.accountApiBaseUrlLabel)).toHaveValue(
      'https://api.idmagic.example/account/acme',
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
