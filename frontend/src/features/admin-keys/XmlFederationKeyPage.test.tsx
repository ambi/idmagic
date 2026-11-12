import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminKey } from '../../types'
import type { RolloverSnapshot } from './xmlFederationRollover'
import { XmlFederationKeyPage } from './XmlFederationKeyPage'
import { xmlFederationKeyDictionary } from './XmlFederationKeyPage.i18n'

const t = xmlFederationKeyDictionary.en

afterEach(() => restoreGlobals())

const active: AdminKey = {
  kid: 'kid-active',
  alg: 'RS256',
  active: true,
  prepared: false,
  created_at: '2026-01-01T00:00:00Z',
  public_jwk: {},
  signing_certificate: 'QUNUSVZF',
  certificate_fingerprint_sha256: 'aa'.repeat(32),
}

const prepared: AdminKey = {
  ...active,
  kid: 'kid-prepared',
  active: false,
  prepared: true,
  signing_certificate: 'TkVYVA==',
  certificate_fingerprint_sha256: 'bb'.repeat(32),
}

function snapshot(nextThumbprint: string, keys: AdminKey[] = [active, prepared]): RolloverSnapshot {
  return {
    keys,
    entraDomains: [
      {
        domain: 'contoso.com',
        verification: { status: 'matched', next_signing_certificate_thumbprint: nextThumbprint },
        updateCommand: "Update-MgDomainFederationConfiguration -DomainId 'contoso.com'",
      },
    ],
    trusts: [
      { id: 'wsfed:urn:app', kind: 'wsfed', label: 'Payroll' },
      { id: 'saml:https://sp.example', kind: 'saml', label: 'Wiki' },
    ],
  }
}

// stubAdminAPI は、画面が読む一覧と詳細を、与えた状態から返す。切り替えの要求は記録する。
function stubAdminAPI(state: () => RolloverSnapshot) {
  const json = (body: unknown, status = 200) => ({
    ok: status < 400,
    status,
    headers: new Headers(),
    json: async () => body,
  })
  const fetchMock = mock(async (url: string, init?: RequestInit) => {
    const current = state()
    if (url.includes('/api/admin/v1/keys/rotate')) {
      const next = current.keys.find((key) => key.prepared) ?? active
      return json({ next: { ...next, active: true, prepared: false } })
    }
    if (url.includes('/api/admin/v1/keys')) return json({ keys: current.keys })
    if (url.includes('/entra-federation/domains/')) {
      const domain = current.entraDomains[0]
      return json({
        domain: { domain: domain.domain },
        verification: domain.verification,
        commands: { create: '', update: domain.updateCommand },
      })
    }
    if (url.includes('/entra-federation/domains')) {
      return json({ domains: current.entraDomains.map((d) => ({ domain: d.domain })) })
    }
    if (url.includes('/wsfed/relying-parties')) {
      return json({ relying_parties: [{ wtrealm: 'urn:app', display_name: 'Payroll' }] })
    }
    if (url.includes('/saml/service-providers')) {
      return json({
        service_providers: [
          { entity_id: 'https://sp.example', idp_profile_id: 'default', display_name: 'Wiki' },
          { entity_id: 'https://other.example', idp_profile_id: 'dedicated' },
        ],
      })
    }
    void init
    return json({}, 404)
  })
  stubGlobal('fetch', fetchMock)
  return fetchMock
}

function activateButton() {
  return screen.getByRole('button', { name: t.activate })
}

function confirmAll() {
  fireEvent.click(screen.getByLabelText(t.trustRegistered.replace('{name}', 'Payroll')))
  fireEvent.click(screen.getByLabelText(t.trustRegistered.replace('{name}', 'Wiki')))
  fireEvent.click(screen.getByLabelText(t.overallConfirm))
}

function rotationCalls(fetchMock: ReturnType<typeof stubAdminAPI>) {
  return fetchMock.mock.calls.filter(([url]) => String(url).includes('/keys/rotate'))
}

describe('XmlFederationKeyPage', () => {
  //spec:covers REQ-SIGNINGKEYS-006: Entra の照合が準備済みの証明書を確かめ、ほかの信頼先と全体を確認するまで切り替えを無効にすること。
  it('keeps activation disabled until every trust confirms the prepared certificate', async () => {
    await renderWithRouter(
      <XmlFederationKeyPage
        csrfToken="csrf"
        snapshot={snapshot(prepared.certificate_fingerprint_sha256!)}
      />,
      { locale: 'en' },
    )
    expect(activateButton()).toBeDisabled()
    fireEvent.click(screen.getByLabelText(t.trustRegistered.replace('{name}', 'Payroll')))
    fireEvent.click(screen.getByLabelText(t.overallConfirm))
    expect(activateButton()).toBeDisabled()
    expect(screen.getByText(t.blockerTrust.replace('{name}', 'Wiki'))).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText(t.trustRegistered.replace('{name}', 'Wiki')))
    expect(activateButton()).toBeEnabled()
  })

  //spec:covers REQ-SIGNINGKEYS-006: 以前の準備済みの鍵で一致した照合を、今の準備済みの鍵の確認として扱わないこと。
  it('does not accept an Entra match that compared an earlier prepared certificate', async () => {
    await renderWithRouter(
      <XmlFederationKeyPage csrfToken="csrf" snapshot={snapshot('cc'.repeat(32))} />,
      { locale: 'en' },
    )
    confirmAll()
    expect(activateButton()).toBeDisabled()
    expect(screen.getByText(t.blockerEntra.replace('{domain}', 'contoso.com'))).toBeInTheDocument()
  })

  //spec:covers REQ-SIGNINGKEYS-006: 確定のときに読み直し、確認した準備済みの kid と trust_updated で切り替えを要求すること。
  it('re-reads the trusts and activates the confirmed prepared kid', async () => {
    const state = snapshot(prepared.certificate_fingerprint_sha256!)
    const fetchMock = stubAdminAPI(() => state)
    await renderWithRouter(<XmlFederationKeyPage csrfToken="csrf" snapshot={state} />, {
      locale: 'en',
    })
    confirmAll()
    fireEvent.click(activateButton())
    fireEvent.click(screen.getAllByRole('button', { name: t.activate }).at(-1)!)
    await waitFor(() => expect(rotationCalls(fetchMock)).toHaveLength(1))
    const [, init] = rotationCalls(fetchMock)[0]
    expect(JSON.parse(String(init?.body))).toEqual({
      usage: 'XmlFederationSigning',
      action: 'activate',
      kid: prepared.kid,
      trust_updated: true,
    })
  })

  //spec:covers REQ-SIGNINGKEYS-006: 確認の後に準備済みの鍵が変われば、切り替えを要求せず確認を外すこと。
  it('refuses to activate when the prepared key changed after the confirmation', async () => {
    const confirmed = snapshot(prepared.certificate_fingerprint_sha256!)
    const replaced = { ...prepared, kid: 'kid-replaced' }
    const fetchMock = stubAdminAPI(() =>
      snapshot(prepared.certificate_fingerprint_sha256!, [active, replaced]),
    )
    await renderWithRouter(<XmlFederationKeyPage csrfToken="csrf" snapshot={confirmed} />, {
      locale: 'en',
    })
    confirmAll()
    fireEvent.click(activateButton())
    fireEvent.click(screen.getAllByRole('button', { name: t.activate }).at(-1)!)
    await waitFor(() => expect(screen.getByText(t.changedSinceConfirmation)).toBeInTheDocument())
    expect(rotationCalls(fetchMock)).toHaveLength(0)
    expect(screen.getByLabelText(t.overallConfirm)).not.toBeChecked()
    expect(activateButton()).toBeDisabled()
  })
})
