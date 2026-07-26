import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminApplicationDetailPage } from './AdminApplicationDetailPage'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { AdminApplication, AdminApplicationDetail } from '../../types'

const t = adminApplicationsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const assignmentFetch = mock((url: string) => {
  if (url.includes('/assignments')) return Promise.resolve(response(200, { assignments: [] }))
  if (url.includes('/users')) return Promise.resolve(response(200, { users: [] }))
  return Promise.resolve(response(200, { groups: [] }))
})

const app: AdminApplication = {
  application_id: 'app-1',
  name: 'Payroll',
  kind: 'federated',
  status: 'active',
  protocol: { type: 'oidc', client_id: 'client-1' },
  category_ids: [],
  category_names: [],
  assigned_subject_count: 0,
  sign_in_policy_summary: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const detail: AdminApplicationDetail = { application: app }

const integrationEndpoints = {
  issuer: 'https://idp.example/realms/acme',
  oauth: {
    openid_configuration: 'https://idp.example/realms/acme/.well-known/openid-configuration',
    oauth_authorization_server:
      'https://idp.example/realms/acme/.well-known/oauth-authorization-server',
    authorization_endpoint: 'https://idp.example/realms/acme/authorize',
    token_endpoint: 'https://idp.example/realms/acme/token',
    userinfo_endpoint: 'https://idp.example/realms/acme/userinfo',
    jwks_uri: 'https://idp.example/realms/acme/jwks',
    revocation_endpoint: 'https://idp.example/realms/acme/revoke',
    introspection_endpoint: 'https://idp.example/realms/acme/introspect',
    end_session_endpoint: 'https://idp.example/realms/acme/end_session',
    registration_endpoint: 'https://idp.example/realms/acme/register',
    pushed_authorization_request_endpoint: 'https://idp.example/realms/acme/par',
    device_authorization_endpoint: 'https://idp.example/realms/acme/device_authorization',
  },
  saml: {
    entity_id: 'https://idp.example/realms/acme',
    metadata_url: 'https://idp.example/realms/acme/saml/metadata',
    sso_url: 'https://idp.example/realms/acme/saml/sso',
    slo_url: 'https://idp.example/realms/acme/saml/slo',
    signing_certificate: {
      download_url: 'https://idp.example/realms/acme/saml/signing-certificate.pem',
      fingerprint_sha256: 'AA:BB:CC',
      not_before: '2026-01-01T00:00:00Z',
      not_after: '2036-01-01T00:00:00Z',
    },
  },
  ws_federation: {
    realm: 'https://idp.example/realms/acme',
    metadata_url:
      'https://idp.example/realms/acme/federationmetadata/2007-06/federationmetadata.xml',
    passive_logon_url: 'https://idp.example/realms/acme/wsfed',
    active_logon_url: 'https://idp.example/realms/acme/trust/usernamemixed',
    metadata_exchange_url: 'https://idp.example/realms/acme/trust/mex',
  },
  apis: {
    management_api_base_url: 'https://idp.example/realms/acme/api/admin',
    scim_base_url: 'https://idp.example/realms/acme/scim/v2',
    account_api_base_url: 'https://idp.example/realms/acme/api/account',
  },
}

describe('AdminApplicationDetailPage', () => {
  afterEach(() => restoreGlobals())

  it('shows an error and keeps the confirmation open when deletion fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'Could not delete the application.' }))),
    )
    await renderWithRouter(
      <AdminApplicationDetailPage
        csrfToken="csrf"
        detail={detail}
        integrationEndpoints={integrationEndpoints}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText('Could not delete the application.')).toBeInTheDocument()
  })

  it('shows OIDC RP setup guidance separately from registered application settings', async () => {
    stubGlobal('fetch', assignmentFetch)
    const oidcDetail: AdminApplicationDetail = {
      application: app,
      oidc: {
        client_id: 'client-1',
        client_type: 'confidential',
        redirect_uris: ['https://rp.example/callback'],
        grant_types: ['authorization_code'],
        response_types: ['code'],
        token_endpoint_auth_method: 'client_secret_basic',
        scope: 'openid profile',
        require_pushed_authorization_requests: false,
        dpop_bound_access_tokens: false,
        fapi_profile: '',
        client_secret_rotatable: true,
        secret_credentials: [],
      },
    }
    await renderWithRouter(
      <AdminApplicationDetailPage
        csrfToken="csrf"
        detail={oidcDetail}
        integrationEndpoints={integrationEndpoints}
      />,
    )
    expect(screen.getByRole('heading', { name: t.idmagicSetupHeading })).toBeInTheDocument()
    expect(screen.getByText(integrationEndpoints.oauth.openid_configuration)).toBeInTheDocument()
    expect(screen.getAllByText('client-1').length).toBeGreaterThan(0)
    expect(screen.queryByText(/client secret/i)).not.toBeInTheDocument()
  })

  it('shows SAML SP setup guidance and certificate download', async () => {
    stubGlobal('fetch', assignmentFetch)
    const samlApp = { ...app, protocol: { type: 'saml' as const, entity_id: 'https://sp.example' } }
    const samlDetail: AdminApplicationDetail = {
      application: samlApp,
      saml: {
        entity_id: 'https://sp.example',
        acs_urls: ['https://sp.example/acs'],
        slo_url: '',
        audience: '',
        name_id_format: 'urn:oasis:names:tc:SAML:2.0:nameid-format:persistent',
        name_id_source: 'sub',
        sign_assertion: true,
        sign_response: false,
        want_authn_requests_signed: false,
        authn_request_signing_certificate_pem: '',
        rules: [],
      },
    }
    await renderWithRouter(
      <AdminApplicationDetailPage
        csrfToken="csrf"
        detail={samlDetail}
        integrationEndpoints={integrationEndpoints}
      />,
    )
    expect(screen.getByText(integrationEndpoints.saml.metadata_url)).toBeInTheDocument()
    expect(screen.getByText(integrationEndpoints.saml.entity_id)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: t.downloadSigningCertificate })).toHaveAttribute(
      'href',
      integrationEndpoints.saml.signing_certificate.download_url,
    )
  })
})
