import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import type { EntraFederationDomainDetail } from '../../api'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminEntraFederationDomainPage } from './AdminEntraFederationDomainPage'
import { adminEntraFederationDictionary } from './AdminEntraFederationPage.i18n'

function detail(overrides: Partial<EntraFederationDomainDetail> = {}): EntraFederationDomainDetail {
  return {
    domain: { domain: 'contoso.com', issuer_uri: 'urn:contoso', entra_tenant_id: 'tenant-guid' },
    profile: {
      domains: [{ domain: 'contoso.com', issuer_uri: 'urn:contoso' }],
      source_anchor_attribute: 'on_premises_immutable_id',
      immutable_id_attribute: 'entra_immutable_id',
    },
    reply_url: 'https://login.microsoftonline.com/login.srf',
    powershell: { IssuerUri: 'urn:contoso', PreferredAuthenticationProtocol: 'wsFed' },
    commands: {
      create: 'New-MgDomainFederationConfiguration X',
      update: 'Update-MgDomainFederationConfiguration X',
    },
    procedure: {
      connect: 'Connect-MgGraph -TenantId tenant-guid',
      connect_read_only: 'Connect-MgGraph -ReadOnly',
      backup: 'Backup-Step',
      compare: 'Compare-Step',
      apply: 'Apply-Step',
      read_back: 'ReadBack-Step',
    },
    certificates: { signing: { sha256_thumbprint: 'abc123', not_after: '2027-01-01T00:00:00Z' } },
    verification: { status: 'unverified' },
    known_limitations: [{ code: 'hybrid_join_not_supported' }],
    ...overrides,
  }
}

const observation = JSON.stringify({ format: 'idmagic.entra-federation-observation.v1' })

describe('ドメインの詳細', () => {
  afterEach(() => restoreGlobals())

  //spec:covers REQ-WSFEDERATION-001: 詳細が変更の操作なしに保存した設定、期待値、証明書、手順とコマンドを示し、それぞれをコピーでき、制限を選択した言語で示し、編集への入口を置くこと。
  for (const locale of ['ja', 'en'] as const) {
    it(`${locale}: shows the saved settings, procedure and commands for copying`, async () => {
      const dictionary = adminEntraFederationDictionary[locale]
      const copy = mock(async (_value: string) => {})
      stubGlobal('navigator', { clipboard: { writeText: copy } })
      await renderWithRouter(
        <AdminEntraFederationDomainPage
          csrfToken="csrf"
          detail={detail({
            unfederated_source_users: { user_count: 1234, sample_usernames: ['frank'] },
          })}
        />,
        { locale },
      )
      expect(screen.getByText(dictionary.statusUnverified)).toBeInTheDocument()
      expect(screen.getByText(dictionary.statusUnverifiedDescription)).toBeInTheDocument()
      expect(screen.getByText(dictionary.verificationManualNotice)).toBeInTheDocument()
      expect(screen.getByText('tenant-guid')).toBeInTheDocument()
      expect(screen.getByText('abc123')).toBeInTheDocument()
      expect(screen.getByText(dictionary.hybridJoinNotProvidedNotice)).toBeInTheDocument()
      expect(screen.queryByText('hybrid_join_not_supported')).not.toBeInTheDocument()
      expect(screen.getByText(dictionary.prerequisiteScp)).toBeInTheDocument()
      expect(screen.getByText(dictionary.postCheckPrt)).toBeInTheDocument()
      expect(
        screen.getByText(
          dictionary.unfederatedSourceUsersNotice.replace('{count}', (1234).toLocaleString(locale)),
          { exact: false },
        ),
      ).toBeInTheDocument()
      expect(screen.getByText(dictionary.edit).closest('a')).toHaveAttribute(
        'href',
        '/admin/federation/entra/contoso.com/edit',
      )
      fireEvent.click(screen.getByRole('button', { name: dictionary.copyCreateCommand }))
      await waitFor(() =>
        expect(copy).toHaveBeenLastCalledWith('New-MgDomainFederationConfiguration X'),
      )
      fireEvent.click(screen.getByRole('button', { name: dictionary.copyUpdateCommand }))
      await waitFor(() =>
        expect(copy).toHaveBeenLastCalledWith('Update-MgDomainFederationConfiguration X'),
      )
      fireEvent.click(
        screen.getByRole('button', { name: `${dictionary.copy}: ${dictionary.procedureReadBack}` }),
      )
      await waitFor(() => expect(copy).toHaveBeenLastCalledWith('ReadBack-Step'))
      expect(screen.getByRole('status')).toHaveTextContent(dictionary.commandCopiedNotice)
    })
  }

  //spec:covers REQ-WSFEDERATION-001: 不一致の差分、取得時刻、確認時刻、確認したテナントを示すこと。
  it('shows the differences and the times of a mismatched check', async () => {
    const t = adminEntraFederationDictionary.en
    await renderWithRouter(
      <AdminEntraFederationDomainPage
        csrfToken="csrf"
        detail={detail({
          verification: {
            status: 'mismatched',
            retrieved_at: '2026-11-11T00:00:00Z',
            checked_at: '2026-11-11T00:01:00Z',
            observed_entra_tenant_id: 'observed-tenant',
            differences: [{ field: 'issuer_uri', expected: 'urn:contoso', observed: 'urn:other' }],
          },
        })}
      />,
    )
    expect(screen.getByText(t.statusMismatched)).toBeInTheDocument()
    expect(screen.getByText('observed-tenant')).toBeInTheDocument()
    expect(screen.getByText('issuer_uri')).toBeInTheDocument()
    expect(screen.getByText('urn:other')).toBeInTheDocument()
    expect(screen.getByText(t.checkedAtLabel)).toBeInTheDocument()
  })
})

describe('観測文書の取り込み', () => {
  afterEach(() => restoreGlobals())
  const t = adminEntraFederationDictionary.en

  //spec:covers REQ-WSFEDERATION-001: 取り込みが照合の結果で状態を置き換え、成功を通知すること。
  it('replaces the status with the recorded check', async () => {
    const fetch = mock(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response(
          JSON.stringify({
            verification: {
              status: 'matched',
              checked_at: '2026-11-11T00:00:00Z',
              differences: [],
            },
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          },
        ),
    )
    stubGlobal('fetch', fetch)
    await renderWithRouter(<AdminEntraFederationDomainPage csrfToken="csrf" detail={detail()} />)
    fireEvent.change(screen.getByLabelText(t.observationLabel), { target: { value: observation } })
    fireEvent.click(screen.getByRole('button', { name: t.importObservation }))
    await waitFor(() => expect(screen.getByText(t.statusMatched)).toBeInTheDocument())
    expect(String(fetch.mock.calls[0][0])).toContain(
      '/api/admin/v1/wsfed/entra-federation/domains/contoso.com/observations',
    )
    expect(JSON.parse(String(fetch.mock.calls[0][1]?.body))).toEqual(JSON.parse(observation))
    expect(screen.getByRole('status')).toHaveTextContent(t.observationRecordedNotice)
  })

  //spec:covers REQ-WSFEDERATION-001: 拒否された観測文書を入力に残して失敗を表示し、状態を変えないこと。
  for (const [code, message] of [
    ['observation_outdated', t.observationOutdatedError],
    ['invalid_request', t.observationRejectedError.replace('{detail}', '')],
  ] as const) {
    it(`keeps the document when the import is refused with ${code}`, async () => {
      stubGlobal(
        'fetch',
        mock(
          async () =>
            new Response(
              JSON.stringify({ type: `urn:idmagic:error:${code}`, status: 400, detail: 'bad' }),
              {
                status: code === 'observation_outdated' ? 409 : 400,
                headers: { 'Content-Type': 'application/problem+json' },
              },
            ),
        ),
      )
      await renderWithRouter(<AdminEntraFederationDomainPage csrfToken="csrf" detail={detail()} />)
      const input = screen.getByLabelText(t.observationLabel)
      fireEvent.change(input, { target: { value: observation } })
      fireEvent.click(screen.getByRole('button', { name: t.importObservation }))
      await waitFor(() =>
        expect(
          within(screen.getByTestId('entra-verification')).getByRole('alert'),
        ).toHaveTextContent(message),
      )
      expect(input).toHaveValue(observation)
      expect(screen.getByText(t.statusUnverified)).toBeInTheDocument()
    })
  }

  it('does not send text that is not JSON', async () => {
    const fetch = mock(() => Promise.resolve(new Response('{}')))
    stubGlobal('fetch', fetch)
    await renderWithRouter(<AdminEntraFederationDomainPage csrfToken="csrf" detail={detail()} />)
    const input = screen.getByLabelText(t.observationLabel)
    fireEvent.change(input, { target: { value: '{"format":' } })
    expect(input.getAttribute('aria-invalid')).toBe('true')
    expect(screen.getByRole('button', { name: t.importObservation })).toBeDisabled()
    expect(fetch).not.toHaveBeenCalled()
  })
})

describe('ドメインの解除', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())
  const t = adminEntraFederationDictionary.en
  const twoDomains = {
    domains: [
      { domain: 'contoso.com', issuer_uri: 'urn:contoso' },
      { domain: 'fabrikam.com', issuer_uri: 'urn:fabrikam' },
    ],
    source_anchor_attribute: 'on_premises_immutable_id',
    immutable_id_attribute: 'entra_immutable_id',
  }

  //spec:covers REQ-WSFEDERATION-001: 解除の項目が残るドメイン、Managed への切り替えのコマンド、以前の構成の復元の説明、移行の手順への入口を示し、観測の取り込みなしに解除を要求できること。最後のドメインでは RP の削除を示すこと。
  it('shows the remaining domains and both ways to roll back', async () => {
    const fetch = mock(() => Promise.resolve(new Response('{}')))
    stubGlobal('fetch', fetch)
    await renderWithRouter(
      <AdminEntraFederationDomainPage csrfToken="csrf" detail={detail({ profile: twoDomains })} />,
    )
    const section = within(screen.getByTestId('entra-removal'))
    expect(
      section.getByText(t.removalRemainingDomains.replace('{domains}', 'fabrikam.com')),
    ).toBeInTheDocument()
    expect(
      section.getByText(/Update-MgDomain -DomainId 'contoso.com' -AuthenticationType 'Managed'/),
    ).toBeInTheDocument()
    expect(section.getByText(t.removalPreviousIdpNotice)).toBeInTheDocument()
    expect(section.getByText(t.removalMigrationGuide).closest('a')).toHaveAttribute(
      'href',
      'https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/migrate-from-federation-to-cloud-authentication',
    )
    expect(section.getByRole('button', { name: t.removeDomain })).toBeEnabled()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('marks the last domain as deleting the relying party', async () => {
    await renderWithRouter(<AdminEntraFederationDomainPage csrfToken="csrf" detail={detail()} />)
    expect(
      within(screen.getByTestId('entra-removal')).getByText(t.removalLastDomain),
    ).toBeInTheDocument()
  })

  it('quotes a stored domain as one PowerShell argument', async () => {
    await renderWithRouter(
      <AdminEntraFederationDomainPage
        csrfToken="csrf"
        detail={detail({
          domain: { domain: "contoso'; Write-Host 'x", issuer_uri: 'urn:contoso' },
        })}
      />,
    )
    expect(
      within(screen.getByTestId('entra-removal')).getByText(
        "Update-MgDomain -DomainId 'contoso''; Write-Host ''x' -AuthenticationType 'Managed'",
        { exact: false },
      ),
    ).toBeInTheDocument()
  })

  //spec:covers REQ-WSFEDERATION-001: 確認の画面で対象と残るドメインを示した後に DELETE で解除を要求し、成功したら一覧へ移ること。
  it('confirms the target and the remaining domains, then deletes and moves to the list', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const fetch = mock(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response(
          JSON.stringify({
            domain: 'contoso.com',
            remaining_domains: ['fabrikam.com'],
            relying_party_deleted: false,
            confirmed_by: 'realm_discovery',
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
    )
    stubGlobal('fetch', fetch)
    await renderWithRouter(
      <AdminEntraFederationDomainPage csrfToken="csrf" detail={detail({ profile: twoDomains })} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.removeDomain }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent(t.removalConfirmTitle.replace('{domain}', 'contoso.com'))
    expect(dialog).toHaveTextContent(t.removalRemainingDomains.replace('{domains}', 'fabrikam.com'))
    fireEvent.click(within(dialog).getByRole('button', { name: t.removeDomain }))
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/federation/entra'),
    )
    expect(String(fetch.mock.calls[0][0])).toMatch(
      /\/api\/admin\/v1\/wsfed\/entra-federation\/domains\/contoso\.com$/,
    )
    expect(fetch.mock.calls[0][1]?.method).toBe('DELETE')
  })

  //spec:covers REQ-WSFEDERATION-001: 拒否された解除で詳細に留まり、まだ IdMagic へ送っていることと確かめられなかったことを、それぞれ辞書の文言で示すこと。
  for (const [code, message] of [
    ['entra_federation_still_active', t.removalStillActiveError],
    ['entra_rollback_unconfirmed', t.removalUnconfirmedError],
  ] as const) {
    it(`stays on the detail when the removal is refused with ${code}`, async () => {
      stubGlobal('location', { ...originalLocation, assign: mock() })
      stubGlobal(
        'fetch',
        mock(
          async () =>
            new Response(
              JSON.stringify({ type: `urn:idmagic:error:${code}`, status: 422, detail: 'refused' }),
              { status: 422, headers: { 'Content-Type': 'application/problem+json' } },
            ),
        ),
      )
      await renderWithRouter(
        <AdminEntraFederationDomainPage
          csrfToken="csrf"
          detail={detail({ profile: twoDomains })}
        />,
      )
      fireEvent.click(screen.getByRole('button', { name: t.removeDomain }))
      const dialog = await screen.findByRole('dialog')
      fireEvent.click(within(dialog).getByRole('button', { name: t.removeDomain }))
      await waitFor(() =>
        expect(within(screen.getByTestId('entra-removal')).getByRole('alert')).toHaveTextContent(
          message,
        ),
      )
      expect(window.location.assign).not.toHaveBeenCalled()
    })
  }
})
