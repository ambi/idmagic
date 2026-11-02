import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { AdminAgentEditPage } from './AdminAgentEditPage'
import { AdminAgentCredentialsPage } from './AdminAgentCredentialsPage'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { parsePublicKey } from './publicKey'
import type { AdminAgent, McpResourceServer } from '../../types'

const t = adminAgentsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const agent: AdminAgent = {
  id: 'agent-1',
  tenant_id: 'tenant-1',
  name: 'invoice-bot',
  kind: 'autonomous',
  owner_user_id: 'user-1',
  status: 'active',
  roles: ['invoice:read'],
  client_ids: [],
  created_at: '2026-01-01T00:00:00Z',
}

const originalLocation = window.location
afterEach(() => {
  restoreGlobals()
  sessionStorage.clear()
})

describe('AdminAgentEditPage', () => {
  it('updates the agent and redirects to its detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { ...agent, name: 'renamed-bot' }))),
    )
    await renderWithRouter(<AdminAgentEditPage csrfToken="csrf" agent={agent} />)

    expect(screen.queryByLabelText(t.bindClientIdAria)).toBeNull()
    expect(screen.queryByText('Guardrails')).toBeNull()

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), {
      target: { value: 'renamed-bot' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/agents/agent-1'),
    )
  })

  it('shows an error and keeps the form when updating fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Could not update the agent.' }))),
    )
    await renderWithRouter(<AdminAgentEditPage csrfToken="csrf" agent={agent} />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), {
      target: { value: 'renamed-bot' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the agent.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

describe('AdminAgentCredentialsPage', () => {
  const publicJWK = { kty: 'RSA', kid: 'k1', n: 'abc', e: 'AQAB' }

  function renderKeyForm() {
    return renderWithRouter(
      <AdminAgentCredentialsPage
        csrfToken="csrf"
        agent={agent}
        resourceServers={[
          { id: 'rs-1', name: 'Docs', scopes: ['tools:read'] } as unknown as McpResourceServer,
        ]}
      />,
    )
  }

  function submitKey(value: string) {
    fireEvent.change(screen.getByLabelText(t.publicKeyLabel, { exact: false }), {
      target: { value },
    })
    fireEvent.click(screen.getByLabelText(/tools:read/))
    fireEvent.click(screen.getByRole('button', { name: t.createKeyCredential }))
  }

  //spec:covers REQ-IDMANAGEMENT-103: 公開鍵から鍵認証のクライアントを作り、その Agent に束縛する。
  it('creates a key client from a public key and binds it', async () => {
    const fetchMock = mock((url: string, _init?: RequestInit) =>
      Promise.resolve(
        url.endsWith('/clients')
          ? response(201, { client: { client_id: 'new-client' } })
          : url.endsWith('/credentials')
            ? response(204)
            : response(200, { ...agent, client_ids: ['new-client'] }),
      ),
    )
    stubGlobal('fetch', fetchMock)
    await renderKeyForm()

    submitKey(JSON.stringify(publicJWK))

    expect(await screen.findByText('new-client')).toBeInTheDocument()
    const call = fetchMock.mock.calls.find(([url]) => url.endsWith('/clients'))
    const body = JSON.parse(String(call?.[1]?.body))
    expect(body.token_endpoint_auth_method).toBe('private_key_jwt')
    expect(body.grant_types).toEqual(['client_credentials', 'urn:openid:params:grant-type:ciba'])
    expect(body.jwks).toEqual({ keys: [publicJWK] })
    expect(body.scope).toBe('openid profile tools:read')
  })

  it('prevents another submission while parsing the public key', async () => {
    const fetchMock = mock((url: string) =>
      Promise.resolve(
        url.endsWith('/clients')
          ? response(201, { client: { client_id: 'new-client' } })
          : url.endsWith('/credentials')
            ? response(204)
            : response(200, { ...agent, client_ids: ['new-client'] }),
      ),
    )
    stubGlobal('fetch', fetchMock)
    await renderKeyForm()
    const input = screen.getByLabelText(t.publicKeyLabel, { exact: false })
    fireEvent.change(input, { target: { value: JSON.stringify(publicJWK) } })
    const form = input.closest('form')!

    fireEvent.submit(form)
    fireEvent.submit(form)

    expect(await screen.findByText('new-client')).toBeInTheDocument()
    expect(fetchMock.mock.calls.filter(([url]) => url.endsWith('/clients'))).toHaveLength(1)
  })

  //spec:covers REQ-IDMANAGEMENT-103: PEM の公開鍵を、アサーションの検証が使える JWK へ変換する。
  it('accepts a PEM public key', async () => {
    const pair = await crypto.subtle.generateKey(
      {
        name: 'RSA-PSS',
        modulusLength: 2048,
        publicExponent: new Uint8Array([1, 0, 1]),
        hash: 'SHA-256',
      },
      true,
      ['sign', 'verify'],
    )
    const der = new Uint8Array(await crypto.subtle.exportKey('spki', pair.publicKey))
    const pem = `-----BEGIN PUBLIC KEY-----\n${btoa(String.fromCharCode(...der))}\n-----END PUBLIC KEY-----\n`
    const parsed = await parsePublicKey(pem)
    expect(parsed).toMatchObject({ keys: [{ kty: 'RSA', alg: 'PS256', e: 'AQAB' }] })
    expect(
      await parsePublicKey('-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----'),
    ).toBe('private')
    expect(
      await parsePublicKey('-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----'),
    ).toBe('unsupported')
  })

  //spec:covers REQ-IDMANAGEMENT-103: 秘密鍵の要素と読めない入力はサーバーへ送らない。
  it.each([
    [JSON.stringify({ ...publicJWK, d: 'secret' }), t.publicKeyHasPrivateMembers],
    [JSON.stringify({ keys: [{ ...publicJWK, p: 'secret' }] }), t.publicKeyHasPrivateMembers],
    ['not json', t.publicKeyInvalid],
    ['{"keys":[]}', t.publicKeyInvalid],
  ])('does not send %s', async (value, message) => {
    const fetchMock = mock(() => Promise.resolve(response(200)))
    stubGlobal('fetch', fetchMock)
    await renderKeyForm()

    submitKey(value)

    expect(await screen.findByText(message)).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  //spec:covers REQ-IDMANAGEMENT-103: 束縛だけが失敗したら、作成済みの client_id で再試行できる。
  it('keeps the created client id when binding fails', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) =>
        Promise.resolve(
          url.endsWith('/clients')
            ? response(201, { client: { client_id: 'new-client' } })
            : response(409, { message: 'bind failed' }),
        ),
      ),
    )
    await renderKeyForm()

    submitKey(JSON.stringify(publicJWK))

    expect(await screen.findByText('bind failed')).toBeInTheDocument()
    expect(screen.getByLabelText(t.bindClientIdAria)).toHaveValue('new-client')
  })

  it('explains why a shared-secret client cannot be bound', async () => {
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(
          response(422, {
            type: 'urn:idmagic:error:agent_client_auth_method_not_allowed',
            detail:
              'An agent can only be bound to a client that authenticates with private_key_jwt or tls_client_auth.',
          }),
        ),
      ),
    )
    await renderWithRouter(<AdminAgentCredentialsPage csrfToken="csrf" agent={agent} />)

    fireEvent.change(screen.getByLabelText(t.bindClientIdAria), {
      target: { value: 'secret-client' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.bind }))

    expect(await screen.findByText(t.agentClientAuthMethodNotAllowedError)).toBeInTheDocument()
    expect(screen.getByLabelText(t.bindClientIdAria)).toHaveValue('secret-client')
  })

  it('shows an error when binding a credential fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'This client_id is already in use.' }))),
    )
    await renderWithRouter(<AdminAgentCredentialsPage csrfToken="csrf" agent={agent} />)

    fireEvent.change(screen.getByLabelText(t.bindClientIdAria), {
      target: { value: 'client-x' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.bind }))

    expect(await screen.findByText('This client_id is already in use.')).toBeInTheDocument()
  })

  it('unbinds a credential', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (init?.method === 'DELETE') return Promise.resolve(response(204))
        if (url.includes('/api/admin/v1/agents/agent-1')) {
          return Promise.resolve(response(200, { ...agent, client_ids: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(
      <AdminAgentCredentialsPage csrfToken="csrf" agent={{ ...agent, client_ids: ['client-a'] }} />,
    )

    expect(screen.getByText('client-a')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.unbind }))
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: t.unbind }),
    )

    await waitFor(() => expect(screen.getByText(t.noCredentialsNotice)).toBeInTheDocument())
  })
})
