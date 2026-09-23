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

// openApiTokensTab は発行済みトークンが無い状態でタブを開き、4 つの具体例が共通に言う
// 「主見出しと一覧の見出しが区別でき、3 つの API の Base URL と用途が表示される」を確かめる。
// 見出しは階層の違いで区別されるので、文言ではなく見出しの水準で読む。
async function openApiTokensTab() {
  const fetchMock = mock().mockResolvedValue(response(200, { tokens: [] }))
  stubGlobal('fetch', fetchMock)
  await renderWithRouter(
    <ApiTokensTab csrfToken="csrf" integrationEndpoints={integrationEndpoints} />,
  )
  await screen.findByText(t.noTokensNotice)

  expect(screen.getByRole('heading', { level: 2, name: t.apiTokensHeading })).toBeInTheDocument()
  expect(
    screen.getByRole('heading', { level: 3, name: t.apiTokensListHeading }),
  ).toBeInTheDocument()
  for (const [label, url, help] of [
    [
      t.managementApiBaseUrlLabel,
      integrationEndpoints.apis.management_api_base_url,
      t.managementApiHelp,
    ],
    [t.scimBaseUrlLabel, integrationEndpoints.apis.scim_base_url, t.scimConnectorHelp],
    [t.accountApiBaseUrlLabel, integrationEndpoints.apis.account_api_base_url, t.accountApiHelp],
  ]) {
    expect(screen.getByLabelText(label)).toHaveValue(url)
    expect(screen.getByText(help)).toBeInTheDocument()
  }
  return fetchMock
}

// scopeRows はリソース名から、そのリソースの選択肢を並べた行ごとに、選べるスコープ値を返す。
// 同じリソース名が複数の API の種類に現れるので、行は画面の並び順で返す。
function scopeRows(resourceLabel: string): HTMLElement[] {
  return screen.getAllByText(resourceLabel).map((label) => {
    const row = label.parentElement?.parentElement
    if (!row) {
      throw new Error(`scope row for ${resourceLabel} is missing`)
    }
    return row
  })
}

function scopeCheckboxes(row: HTMLElement): string[] {
  return Array.from(row.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')).map(
    (input) => input.value,
  )
}

describe('ApiTokensTab', () => {
  afterEach(() => restoreGlobals())

  //spec:covers EX-APITOKENS-001-01: 見出しの階層と 3 つの Base URL・用途を表示し、スコープを API の種類とリソースでまとめて正式な値と read / write の意味を示し、選んだスコープだけを発行要求へ載せること。
  it('groups scopes by API and resource and issues only the selected ones', async () => {
    const fetchMock = await openApiTokensTab()
    fetchMock
      .mockResolvedValueOnce(
        response(201, {
          token: 'header.payload.signature',
          meta: {
            id: 'token-1',
            description: 'Directory sync',
            scopes: ['groups:read'],
            created_at: '2026-07-23T00:00:00Z',
          },
        }),
      )
      .mockResolvedValueOnce(response(200, { tokens: [] }))

    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
    for (const heading of [
      t.managementScopesHeading,
      t.scimScopesHeading,
      t.accountScopesHeading,
    ]) {
      expect(screen.getByText(heading)).toBeInTheDocument()
    }
    expect(scopeRows(t.groupsScopeResourceLabel).map(scopeCheckboxes)).toEqual([
      ['groups:read', 'groups:write'],
      ['scim:groups:read', 'scim:groups:write'],
    ])
    expect(screen.getByText('groups:read')).toBeInTheDocument()
    expect(screen.getByText(t.readScopeDescription)).toBeInTheDocument()
    expect(screen.getByText(t.writeScopeDescription)).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(t.tokenDescriptionLabel), {
      target: { value: 'Directory sync' },
    })
    fireEvent.click(screen.getByLabelText('groups:read'))
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    expect(await screen.findByDisplayValue('header.payload.signature')).toBeInTheDocument()
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')
    expect(JSON.parse(String(post?.[1]?.body)).scopes).toEqual(['groups:read'])
  })

  //spec:covers EX-APITOKENS-001-02: 共通の見出しと Base URL を表示したうえで、3 つのスコープグループがすべて閉じた状態で始まり、別のグループのスコープを選んでも必要としないグループは閉じたままであること。
  it('keeps scope groups the administrator does not need collapsed', async () => {
    await openApiTokensTab()
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    const groups = [t.managementScopesHeading, t.scimScopesHeading, t.accountScopesHeading].map(
      (heading) => screen.getByText(heading).closest('details'),
    )
    for (const group of groups) {
      expect(group).not.toHaveAttribute('open')
    }

    fireEvent.click(screen.getByLabelText('scim:users:read'))
    expect(screen.getByLabelText('scim:users:read').querySelector('input')).toBeChecked()
    expect(groups[0]).not.toHaveAttribute('open')
    expect(groups[2]).not.toHaveAttribute('open')
  })

  //spec:covers EX-APITOKENS-001-03: 共通の見出しと Base URL を表示したうえで、変更系スコープを持たない監査ログのリソースには参照の選択肢だけがあり、存在しない audit:write を選択肢に出さないこと。
  it('offers no write choice for a resource without a write scope', async () => {
    await openApiTokensTab()
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    expect(scopeRows(t.auditScopeResourceLabel).map(scopeCheckboxes)).toEqual([['audit:read']])
    expect(screen.queryByLabelText('audit:write')).not.toBeInTheDocument()
  })

  //spec:covers EX-APITOKENS-001-04: 共通の見出しと Base URL を表示したうえで、変更系スコープだけを持つ 4 つのアカウントのリソースがどれも参照の選択肢を持たず、その変更系スコープを右の変更列へ置くこと。
  it('places a write-only scope in the write column and leaves the read column empty', async () => {
    await openApiTokensTab()
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    for (const [resourceLabel, scope] of [
      [t.accountMfaScopeLabel, 'account:mfa:write'],
      [t.accountSessionsScopeLabel, 'account:sessions:write'],
      [t.accountConsentsScopeLabel, 'account:consents:write'],
      [t.accountPasswordScopeLabel, 'account:password:write'],
    ]) {
      // セッションと同意は管理 API にも同じリソース名がある。アカウントのグループは最後に並ぶ。
      const accountRow = scopeRows(resourceLabel).at(-1)
      expect(accountRow && scopeCheckboxes(accountRow)).toEqual([scope])
      expect(screen.getByLabelText(scope)).toHaveClass('sm:col-start-2')
    }
    // 参照と変更の両方を持つリソースは左の列から並べる。列指定が常に付く実装と区別する。
    expect(screen.getByLabelText('account:write')).not.toHaveClass('sm:col-start-2')
  })

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
