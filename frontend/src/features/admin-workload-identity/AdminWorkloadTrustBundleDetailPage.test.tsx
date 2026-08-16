import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import type { AdminAgent, AgentWorkloadBinding, WorkloadTrustBundle } from '../../types'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { AdminWorkloadTrustBundleDetailPage } from './AdminWorkloadTrustBundleDetailPage'

const t = adminWorkloadIdentityDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(),
})

const bundle: WorkloadTrustBundle = {
  id: 'bundle-1',
  tenant_id: 'tenant-a',
  name: 'prod-cluster',
  trust_domain: 'example.org',
  issuer: 'https://issuer.example',
  jwks_uri: 'https://issuer.example/keys',
  has_inline_jwks: false,
  accepted_audiences: ['idmagic'],
  max_subject_token_ttl_seconds: 3600,
  status: 'enabled',
  created_at: '2026-08-16T00:00:00Z',
}

const binding: AgentWorkloadBinding = {
  id: 'binding-1',
  tenant_id: 'tenant-a',
  trust_bundle_id: 'bundle-1',
  subject_pattern: 'spiffe://example.org/ns/prod/sa/*',
  agent_id: 'agent-1',
  status: 'enabled',
  created_at: '2026-08-16T00:00:00Z',
}

const agent: AdminAgent = {
  id: 'agent-1',
  tenant_id: 'tenant-a',
  name: 'checkout-bot',
  kind: 'autonomous',
  owner_user_id: 'user-1',
  status: 'active',
  roles: [],
  client_ids: [],
  created_at: '2026-08-16T00:00:00Z',
}

function renderDetail(
  overrides: Partial<Parameters<typeof AdminWorkloadTrustBundleDetailPage>[0]>,
) {
  return renderWithRouter(
    <AdminWorkloadTrustBundleDetailPage
      csrfToken="csrf"
      trustBundle={bundle}
      bindings={[binding]}
      agents={[agent]}
      rejectionEvents={[]}
      {...overrides}
    />,
  )
}

describe('AdminWorkloadTrustBundleDetailPage', () => {
  afterEach(() => restoreGlobals())

  it('reads the subject pattern against the agent it maps to, by name', async () => {
    await renderDetail({})

    expect(screen.getByText('spiffe://example.org/ns/prod/sa/*')).toBeInTheDocument()
    // picker の現在値にも同じ名前が出るため、行と合わせて 2 箇所に現れる。
    expect(screen.getAllByText('checkout-bot').length).toBeGreaterThan(0)
  })

  it('falls back to the raw agent id when the agent is outside the picker page', async () => {
    await renderDetail({ agents: [] })
    expect(screen.getByText('agent-1')).toBeInTheDocument()
  })

  it('warns when the bound agent has more than one credential binding', async () => {
    // 交換は最初の資格情報バインディングを採るが、その規則は仕様に無い。決めないまま
    // 「どれが使われるか定まらない」という状態そのものを見せる。
    await renderDetail({ agents: [{ ...agent, client_ids: ['client-a', 'client-b'] }] })
    expect(screen.getByText(t.multipleCredentialsWarning)).toBeInTheDocument()
  })

  it('stays quiet when the bound agent has exactly one credential binding', async () => {
    await renderDetail({ agents: [{ ...agent, client_ids: ['client-a'] }] })
    expect(screen.queryByText(t.multipleCredentialsWarning)).not.toBeInTheDocument()
  })

  it('does not claim there were no rejections when the fetched window was full', async () => {
    await renderDetail({ rejectionEvents: [], rejectionsTruncated: true })

    expect(screen.getByText(t.rejectionsTruncatedEmptyNotice)).toBeInTheDocument()
    expect(screen.queryByText(t.rejectionsEmptyNotice)).not.toBeInTheDocument()
  })

  it('does not delete the bundle until the cascade impact has been confirmed', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) => Promise.resolve(response(204)))
    stubGlobal('fetch', fetchMock)
    await renderDetail({})

    fireEvent.click(screen.getByRole('button', { name: t.deleteBundleTitle }))

    // 確認が出るまで削除は走らない。カスケードで消えるバインディングの件数が本文に出る。
    expect(fetchMock).not.toHaveBeenCalled()
    expect(
      screen.getByText(
        t.deleteBundleMessage.replace('{name}', 'prod-cluster').replace('{count}', '1'),
      ),
    ).toBeInTheDocument()
  })

  it('drops the binding clause from the confirmation when nothing cascades', async () => {
    await renderDetail({ bindings: [] })

    fireEvent.click(screen.getByRole('button', { name: t.deleteBundleTitle }))

    expect(
      screen.getByText(t.deleteBundleMessageNoBindings.replace('{name}', 'prod-cluster')),
    ).toBeInTheDocument()
  })

  it('abandons the delete when the confirmation is cancelled', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) => Promise.resolve(response(204)))
    stubGlobal('fetch', fetchMock)
    await renderDetail({})

    fireEvent.click(screen.getByRole('button', { name: t.deleteBundleTitle }))
    fireEvent.click(screen.getByRole('button', { name: t.cancel }))

    expect(fetchMock).not.toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('disables a bundle only after confirming, and reports the new state', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) => Promise.resolve(response(204)))
    stubGlobal('fetch', fetchMock)
    await renderDetail({})

    fireEvent.click(screen.getByRole('button', { name: t.disableBundleTitle }))
    expect(fetchMock).not.toHaveBeenCalled()

    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: t.disable }))

    expect(
      await screen.findByText(t.disabledNotice.replace('{name}', 'prod-cluster')),
    ).toBeInTheDocument()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain('/api/admin/v1/workload-identity/trust-bundles/bundle-1/disable')
    expect(init.method).toBe('POST')
  })

  it('reports an unreachable JWKS instead of claiming the refresh succeeded', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { reachable: false }))),
    )
    await renderDetail({})

    fireEvent.click(screen.getByRole('button', { name: new RegExp(t.refreshJwks) }))

    expect(await screen.findByText(t.jwksUnreachableError)).toBeInTheDocument()
  })

  it('reports how many keys the refresh actually fetched', async () => {
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(
          response(200, { reachable: true, key_count: 2, jwks_cached_at: '2026-08-16T04:00:00Z' }),
        ),
      ),
    )
    await renderDetail({})

    fireEvent.click(screen.getByRole('button', { name: new RegExp(t.refreshJwks) }))

    expect(
      await screen.findByText(t.jwksRefreshedNotice.replace('{count}', '2')),
    ).toBeInTheDocument()
  })

  it('refuses to offer binding creation when the tenant has no agents to bind to', async () => {
    await renderDetail({ agents: [], bindings: [] })

    expect(screen.getByText(t.noAgentsNotice)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: new RegExp(t.addBinding) })).toBeDisabled()
  })

  it('creates a binding for the chosen pattern and agent', async () => {
    const fetchMock = mock((url: string, _init: RequestInit) =>
      Promise.resolve(
        url.includes('/bindings') && !url.endsWith('/bindings')
          ? response(200, { bindings: [binding] })
          : response(201, binding),
      ),
    )
    stubGlobal('fetch', fetchMock)
    await renderDetail({ bindings: [] })

    fireEvent.change(screen.getByLabelText(t.subjectPatternLabel), {
      target: { value: 'system:serviceaccount:prod:*' },
    })
    fireEvent.click(screen.getByRole('button', { name: new RegExp(t.addBinding) }))

    expect(await screen.findByText(t.bindingCreatedNotice)).toBeInTheDocument()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain('/api/admin/v1/workload-identity/trust-bundles/bundle-1/bindings')
    expect(JSON.parse(String(init.body))).toEqual({
      subject_pattern: 'system:serviceaccount:prod:*',
      agent_id: 'agent-1',
    })
  })
})
