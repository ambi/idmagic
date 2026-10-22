import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminIntegrationEndpointCatalog } from '../../types'
import {
  AdminEntraFederationAddPage,
  AdminEntraFederationPage,
  EntraFederationList,
} from './AdminEntraFederationPage'
import { adminEntraFederationDictionary } from './AdminEntraFederationPage.i18n'

const t = adminEntraFederationDictionary.en

const integrationEndpoints: AdminIntegrationEndpointCatalog = {
  issuer: 'https://login.idmagic.example/realms/acme',
  oauth: {
    openid_configuration: '',
    oauth_authorization_server: '',
    authorization_endpoint: '',
    token_endpoint: '',
    userinfo_endpoint: '',
    jwks_uri: '',
    revocation_endpoint: '',
    introspection_endpoint: '',
    end_session_endpoint: '',
    registration_endpoint: '',
    pushed_authorization_request_endpoint: '',
    device_authorization_endpoint: '',
  },
  saml: {
    entity_id: '',
    metadata_url: '',
    sso_url: '',
    slo_url: '',
    signing_certificate: {
      download_url: '',
      fingerprint_sha256: '',
      not_before: '',
      not_after: '',
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
  apis: { management_api_base_url: '', scim_base_url: '', account_api_base_url: '' },
}

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('locale', () => {
  afterEach(() => restoreGlobals())

  it('renders the entra federation page in English by default', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        csrfToken="csrf"
        actorUsername="admin"
        relyingParties={[]}
        integrationEndpoints={integrationEndpoints}
      />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()
  })

  it('renders the entra federation page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        csrfToken="csrf"
        actorUsername="admin"
        relyingParties={[]}
        integrationEndpoints={integrationEndpoints}
      />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminEntraFederationDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })

  it('uses catalog URLs for Entra federation endpoints', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        csrfToken="csrf"
        actorUsername="admin"
        relyingParties={[]}
        integrationEndpoints={integrationEndpoints}
      />,
    )

    expect(
      screen.getByText(integrationEndpoints.ws_federation.metadata_url).closest('a'),
    ).toHaveAttribute('href', integrationEndpoints.ws_federation.metadata_url)
    expect(
      screen.getByText(integrationEndpoints.ws_federation.metadata_exchange_url),
    ).toBeInTheDocument()
  })
})

describe('EntraFederationList', () => {
  it('renders an empty state and delegates a deletion action', () => {
    const { rerender } = renderEn(<EntraFederationList items={[]} onDelete={mock()} />)
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()

    const onDelete = mock()
    rerender(
      <LocaleProvider initialLocale="en">
        <EntraFederationList
          items={[
            {
              tenant_id: 'tenant-a',
              wtrealm: 'urn:contoso',
              reply_urls: ['https://contoso.example.test/reply'],
              claim_policy: { name_id: { format: 'persistent', source_attribute: 'sub' } },
              created_at: '2026-01-01T00:00:00Z',
              entra_profile: {
                domains: [
                  { domain: 'contoso.com', issuer_uri: 'urn:idmagic:entra:contoso.com' },
                  {
                    domain: 'fabrikam.com',
                    issuer_uri: 'http://fabrikam.com/adfs/services/trust/',
                  },
                ],
                source_anchor_attribute: 'object_guid',
                immutable_id_attribute: 'object_guid',
              },
            },
          ]}
          onDelete={onDelete}
        />
      </LocaleProvider>,
    )
    // 一つの Entra の RP に構成した、すべてのドメインとその IssuerUri を示す。
    expect(screen.getByText('contoso.com')).toBeInTheDocument()
    expect(screen.getByText('fabrikam.com')).toBeInTheDocument()
    expect(
      screen.getByText(`${t.issuerUriPrefix}http://fabrikam.com/adfs/services/trust/`),
    ).toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', { name: t.deleteAriaLabel.replace('{wtrealm}', 'urn:contoso') }),
    )
    expect(onDelete).toHaveBeenCalledWith(expect.objectContaining({ wtrealm: 'urn:contoso' }))
  })
})

describe('Entra の構成の拒否', () => {
  afterEach(() => restoreGlobals())

  //spec:covers REQ-SYSTEM-011: Entra の構成のエラーコードを辞書の文言へ対応付け、内部の英語を表示しないこと。
  //spec:covers REQ-WSFEDERATION-001: sourceAnchor の値の問題について、応答の件数とユーザー名を表示すること。
  for (const locale of ['ja', 'en'] as const) {
    for (const code of [
      'source_anchor_missing',
      'source_anchor_invalid',
      'source_anchor_not_unique',
      'source_anchor_editable',
      'invalid_request',
      'unexpected_error',
    ]) {
      it(`${locale}: ${code} を翻訳し、値の問題の詳細を示す`, async () => {
        stubGlobal(
          'fetch',
          mock(
            async () =>
              new Response(
                JSON.stringify({
                  type: `urn:idmagic:error:${code}`,
                  status: 422,
                  detail: 'Internal English detail must not be shown',
                  user_count: 7,
                  sample_usernames: ['ann@contoso.com', 'bob@contoso.com'],
                }),
                { status: 422, headers: { 'Content-Type': 'application/problem+json' } },
              ),
          ),
        )
        await renderWithRouter(<AdminEntraFederationAddPage csrfToken="csrf" />, { locale })
        fireEvent.change(
          screen.getByLabelText(adminEntraFederationDictionary[locale].verifiedDomainLabel),
          { target: { value: 'contoso.com' } },
        )
        fireEvent.click(
          screen.getByRole('button', { name: adminEntraFederationDictionary[locale].save }),
        )
        await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
        const alert = screen.getByRole('alert')
        const dictionary = adminEntraFederationDictionary[locale]
        if (code === 'source_anchor_not_unique')
          expect(alert).toHaveTextContent(dictionary.sourceAnchorNotUniqueError)
        if (code === 'source_anchor_editable')
          expect(alert).toHaveTextContent(dictionary.sourceAnchorEditableError)
        if (code === 'source_anchor_missing')
          expect(alert).toHaveTextContent(
            dictionary.sourceAnchorMissingError.replace('{count}', '7'),
          )
        if (code === 'source_anchor_invalid')
          expect(alert).toHaveTextContent(
            dictionary.sourceAnchorInvalidError.replace('{count}', '7'),
          )
        if (code === 'unexpected_error') {
          expect(alert).toHaveTextContent('Internal English detail')
        } else {
          expect(alert).not.toHaveTextContent('Internal English detail')
        }
        if (code === 'source_anchor_missing' || code === 'source_anchor_invalid') {
          expect(alert).toHaveTextContent('7')
          expect(alert).toHaveTextContent('ann@contoso.com')
          expect(alert).toHaveTextContent('bob@contoso.com')
        } else {
          expect(alert).not.toHaveTextContent('ann@contoso.com')
        }
      })
    }
  }
})
