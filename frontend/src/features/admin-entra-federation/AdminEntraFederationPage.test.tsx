import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminIntegrationEndpointCatalog } from '../../types'
import {
  AdminEntraFederationAddPage,
  AdminEntraFederationEditPage,
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
        actorUsername="admin"
        relyingParties={[]}
        domains={[]}
        integrationEndpoints={integrationEndpoints}
      />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()
  })

  it('renders the entra federation page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        actorUsername="admin"
        relyingParties={[]}
        domains={[]}
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
        actorUsername="admin"
        relyingParties={[]}
        domains={[]}
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
  //spec:covers REQ-WSFEDERATION-001: 一覧が Entra の RP 全体を削除する操作を置かないこと。
  it('renders an empty state and offers no deletion of the whole relying party', () => {
    const { rerender } = renderEn(<EntraFederationList items={[]} summaries={[]} />)
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()

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
          summaries={[
            {
              domain: 'contoso.com',
              issuer_uri: 'urn:idmagic:entra:contoso.com',
              verification: { status: 'matched' },
            },
          ]}
        />
      </LocaleProvider>,
    )
    // 一つの Entra の RP に構成した、すべてのドメインとその IssuerUri を示す。
    expect(screen.getByText('contoso.com')).toBeInTheDocument()
    expect(screen.getByText('fabrikam.com')).toBeInTheDocument()
    expect(
      screen.getByText(`${t.issuerUriPrefix}http://fabrikam.com/adfs/services/trust/`),
    ).toBeInTheDocument()
    // 解除は切り戻しを確かめるドメインの詳細だけが行う。
    expect(screen.queryByRole('button')).toBeNull()
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
          expect(alert).toHaveTextContent(
            dictionary.sourceAnchorRepairHint.replace('{attribute}', 'on_premises_immutable_id'),
          )
        } else {
          expect(alert).not.toHaveTextContent('ann@contoso.com')
        }
      })
    }
  }
})

describe('ドメインの一覧の確認の状態', () => {
  //spec:covers REQ-WSFEDERATION-001: 一覧がドメインごとの確認の状態を示し、照合のないドメインを反映済みとして示さないこと。
  it('shows each domain with its verification status and links to its detail', () => {
    renderEn(
      <EntraFederationList
        items={[
          {
            tenant_id: 'tenant-a',
            wtrealm: 'urn:federation:MicrosoftOnline',
            reply_urls: [],
            claim_policy: { name_id: { format: 'persistent', source_attribute: 'sub' } },
            created_at: '2026-01-01T00:00:00Z',
            entra_profile: {
              domains: [
                { domain: 'contoso.com', issuer_uri: 'urn:contoso' },
                { domain: 'fabrikam.com', issuer_uri: 'urn:fabrikam' },
              ],
              source_anchor_attribute: 'object_guid',
              immutable_id_attribute: 'entra_immutable_id',
            },
          },
        ]}
        summaries={[
          { domain: 'fabrikam.com', issuer_uri: 'urn:fabrikam', verification: { status: 'stale' } },
        ]}
      />,
    )
    const contoso = screen.getByRole('link', {
      name: t.viewDomainDetail.replace('{domain}', 'contoso.com'),
    })
    expect(contoso).toHaveAttribute('href', '/admin/federation/entra/contoso.com')
    expect(contoso.closest('li')).toHaveTextContent(t.statusUnverified)
    expect(
      screen
        .getByRole('link', { name: t.viewDomainDetail.replace('{domain}', 'fabrikam.com') })
        .closest('li'),
    ).toHaveTextContent(t.statusStale)
  })
})

describe('保存の後の遷移', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  //spec:covers REQ-WSFEDERATION-001: 保存に成功したらテナント ID と組み込みの sourceAnchor を送り、ドメインの詳細へ移ること。
  it('sends the tenant ID and moves to the domain detail after saving', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const fetch = mock(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response('{}', { status: 201, headers: { 'Content-Type': 'application/json' } }),
    )
    stubGlobal('fetch', fetch)
    await renderWithRouter(<AdminEntraFederationAddPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText(t.verifiedDomainLabel), {
      target: { value: 'Contoso.com' },
    })
    fireEvent.change(screen.getByLabelText(t.entraTenantIdLabel), {
      target: { value: '0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/federation/entra/contoso.com'),
    )
    const body = JSON.parse(String(fetch.mock.calls[0][1]?.body))
    expect(body.source_anchor_attribute).toBe('on_premises_immutable_id')
    expect(body.entra_tenant_id).toBe('0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0')
  })

  it('rejects a tenant ID that is not a GUID without saving', async () => {
    const fetch = mock(() => Promise.resolve(new Response('{}')))
    stubGlobal('fetch', fetch)
    await renderWithRouter(<AdminEntraFederationAddPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText(t.verifiedDomainLabel), {
      target: { value: 'contoso.com' },
    })
    const tenant = screen.getByLabelText(t.entraTenantIdLabel)
    fireEvent.change(tenant, { target: { value: 'contoso' } })
    expect(tenant).toHaveAccessibleDescription(t.entraTenantIdInvalidError)
    expect(screen.getByRole('button', { name: t.save })).toBeDisabled()
    expect(fetch).not.toHaveBeenCalled()
  })
})

describe('Entra の構成の入力', () => {
  afterEach(() => restoreGlobals())

  //spec:covers REQ-WSFEDERATION-001: 通常操作では内部属性の入力を求めず、既存の独自属性を読み取り専用で示して保存すること。
  it('既存の独自属性は詳細設定を開かずに保持して保存する', async () => {
    const fetch = mock(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
    stubGlobal('fetch', fetch)
    stubGlobal('location', { ...window.location, assign: mock() })
    await renderWithRouter(
      <AdminEntraFederationEditPage
        csrfToken="csrf"
        domain={{ domain: 'contoso.com', issuer_uri: 'urn:contoso' }}
        sourceAnchorAttribute="legacy_anchor"
        replyURL="https://login.microsoftonline.com/login.srf"
      />,
    )
    expect(screen.queryByLabelText(t.sourceAnchorAttributeLabel) === null).toBe(true)
    expect(screen.getByText('legacy_anchor')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(t.issuerUriLabel), {
      target: { value: 'urn:contoso:updated' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))
    await waitFor(() => expect(fetch).toHaveBeenCalled())
    const body = JSON.parse(String(fetch.mock.calls[0][1]?.body))
    expect(body.source_anchor_attribute).toBe('legacy_anchor')
    expect(body.issuer_uri).toBe('urn:contoso:updated')
  })

  //spec:covers REQ-WSFEDERATION-001: 詳細設定を明示して変更した保存先は、閉じても保持し、一意性と本人の編集禁止の説明を表示すること。
  it('詳細設定で変更した属性を閉じても保持する', async () => {
    const fetch = mock(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response('{}', { status: 201, headers: { 'Content-Type': 'application/json' } }),
    )
    stubGlobal('fetch', fetch)
    stubGlobal('location', { ...window.location, assign: mock() })
    await renderWithRouter(<AdminEntraFederationAddPage csrfToken="csrf" />)
    expect(screen.queryByLabelText(t.sourceAnchorAttributeLabel) === null).toBe(true)
    fireEvent.click(screen.getByRole('button', { name: t.sourceAnchorAdvancedSettings }))
    expect(screen.getByText(t.sourceAnchorAdvancedHint)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(t.sourceAnchorAttributeLabel), {
      target: { value: 'custom_anchor' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.sourceAnchorAdvancedSettings }))
    expect(screen.queryByLabelText(t.sourceAnchorAttributeLabel) === null).toBe(true)
    expect(screen.getByText('custom_anchor')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(t.verifiedDomainLabel), {
      target: { value: 'contoso.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))
    await waitFor(() => expect(fetch).toHaveBeenCalled())
    expect(JSON.parse(String(fetch.mock.calls[0][1]?.body)).source_anchor_attribute).toBe(
      'custom_anchor',
    )
  })

  //spec:covers WCAG22-LABELS-ERRORS: サーバーが拒む domain と reply URL を欄のエラー文で示し、保存しないこと
  it('rejects a domain written as a URL and a reply URL without a host', async () => {
    const fetchMock = mock(() => Promise.resolve(new Response('{}')))
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(<AdminEntraFederationAddPage csrfToken="csrf" />)

    const domain = screen.getByLabelText(t.verifiedDomainLabel)
    fireEvent.change(domain, { target: { value: 'https://contoso.com' } })
    expect(domain).toHaveAccessibleDescription(t.domainNotUrlError)
    fireEvent.change(domain, { target: { value: 'contoso.com' } })

    const reply = screen.getByLabelText(t.replyUrlLabel)
    fireEvent.change(reply, { target: { value: 'login.srf' } })
    expect(reply.getAttribute('aria-invalid')).toBe('true')
    const save = screen.getByRole('button', { name: t.save })
    expect(save).toBeDisabled()
    fireEvent.submit(save.closest('form') as HTMLFormElement)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
