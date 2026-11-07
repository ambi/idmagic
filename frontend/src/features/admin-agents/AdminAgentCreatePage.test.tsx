import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { AdminAgentCreatePage } from './AdminAgentCreatePage'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { agentCreateFlowDictionary } from './agentCreateFlow.i18n'
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
  client_ids: [],
  created_at: '2026-01-01T00:00:00Z',
}

const flow = agentCreateFlowDictionary.en

const server: McpResourceServer = {
  tenant_id: 'tenant-1',
  id: 'server-1',
  resource: 'https://docs.example/mcp',
  name: 'Docs MCP',
  scopes: ['mcp'],
  state: 'Active',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  access_rules: [
    {
      resource_type: 'idmagic_mcp_operation',
      relation: 'invoke',
      scopes: ['mcp'],
      tool: {
        id: 'op-read',
        name: 'read_document',
        display_name: 'Read documents',
        assignment_resource_id: 'server-1/op-read',
      },
    },
    {
      resource_type: 'idmagic_mcp_operation',
      relation: 'invoke',
      scopes: ['mcp'],
      tool: {
        id: 'op-delete',
        name: 'delete_document',
        display_name: 'Delete documents',
        assignment_resource_id: 'server-1/op-delete',
      },
    },
  ],
}

const publicJWK = '{"kty":"EC","crv":"P-256","x":"x-value","y":"y-value"}'

// routedFetch は登録、作成、束縛、割当の各段階に応答し、呼び出しを記録する。bind が失敗を返す回数を指定できる。
function routedFetch(failBind = 0) {
  const calls: { url: string; body?: unknown }[] = []
  stubGlobal(
    'fetch',
    mock((url: string, init?: RequestInit) => {
      calls.push({ url, body: init?.body ? JSON.parse(String(init.body)) : undefined })
      if (url.endsWith('/api/admin/v1/agents'))
        return Promise.resolve(response(201, { ...agent, id: 'agent-2' }))
      if (url.endsWith('/api/admin/v1/clients'))
        return Promise.resolve(response(201, { client: { client_id: 'agent-client' } }))
      if (url.endsWith('/credentials')) {
        if (failBind > 0) {
          failBind--
          return Promise.resolve(response(500, { message: 'Binding failed.' }))
        }
        return Promise.resolve(response(204))
      }
      return Promise.resolve(response(200, {}))
    }),
  )
  return calls
}

describe('AdminAgentCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('registers an agent and redirects to its detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const calls = routedFetch()
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" />)

    expect(document.querySelector('#agent-roles')).toBeNull()
    fireEvent.change(screen.getByLabelText(t.agentNameLabel), { target: { value: 'billing-bot' } })
    fireEvent.click(screen.getByLabelText(flow.authLater))
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/agents/agent-2'),
    )
    expect(calls).toEqual([
      {
        url: expect.stringContaining('/api/admin/v1/agents'),
        body: { name: 'billing-bot', kind: 'autonomous' },
      },
    ])
  })

  it('shows an error and keeps the form when registration fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'This agent name is already in use.' }))),
    )
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), { target: { value: 'billing-bot' } })
    fireEvent.click(screen.getByLabelText(flow.authLater))
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText('This agent name is already in use.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  //spec:covers REQ-IDMANAGEMENT-105: 一つの送信で Agent、公開鍵のクライアント、束縛、選んだ操作の割当を作り、配置先へ渡す値をまとめて示す。
  it('creates the agent, its key client, the binding, and the chosen assignments in one submission', async () => {
    const calls = routedFetch()
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" resourceServers={[server]} />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), { target: { value: 'docs-bot' } })
    fireEvent.change(screen.getByLabelText(flow.publicKeyLabel), { target: { value: publicJWK } })
    fireEvent.click(screen.getByLabelText('Read documents'))
    expect(document.querySelector('#agent-create-summary')).toHaveTextContent(
      'Register "docs-bot", and let it authenticate with a public key. Allow Read documents.',
    )
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText(flow.resultHeading)).toBeInTheDocument()
    expect(
      calls.map((call) => new URL(call.url, 'http://x').pathname.replace(/^\/realms\/[^/]+/, '')),
    ).toEqual([
      '/api/admin/v1/agents',
      '/api/admin/v1/clients',
      '/api/admin/v1/agents/agent-2/credentials',
      '/api/admin/v1/authorization/relation-tuples',
    ])
    expect(calls[1]?.body).toMatchObject({
      token_endpoint_auth_method: 'private_key_jwt',
      scope: 'openid profile mcp',
      jwks: { keys: [JSON.parse(publicJWK)] },
    })
    expect(calls[3]?.body).toEqual({
      writes: [
        {
          resource_type: 'idmagic_mcp_operation',
          resource_id: 'server-1/op-read',
          relation: 'invoke',
          subject_type: 'agent',
          subject_id: 'agent-2',
        },
      ],
      deletes: [],
    })
    const config = document.querySelector('#result-agent-config')?.textContent ?? ''
    expect(config).toContain('client_id: "agent-client"')
    expect(config).toContain('resource: https://docs.example/mcp')
    expect(config).not.toContain('agent-2')
  })

  //spec:covers REQ-IDMANAGEMENT-105: JWKS の URL を選ぶと jwks_uri のクライアントを作り、公開鍵を求めない。
  it('creates a client that reads keys from a JWKS URL', async () => {
    const calls = routedFetch()
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" resourceServers={[server]} />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), { target: { value: 'docs-bot' } })
    fireEvent.click(screen.getByLabelText(flow.authJwksUri))
    expect(screen.queryByLabelText(flow.publicKeyLabel)).toBeNull()
    fireEvent.change(screen.getByLabelText(flow.jwksUriLabel), {
      target: { value: 'https://agent.example/jwks.json' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText(flow.resultHeading)).toBeInTheDocument()
    expect(calls[1]?.body).toMatchObject({ jwks_uri: 'https://agent.example/jwks.json' })
    expect(calls[1]?.body).not.toHaveProperty('jwks')
  })

  //spec:covers REQ-IDMANAGEMENT-105: 束縛だけが失敗したら、再試行で Agent とクライアントを作り直さず束縛から続ける。
  //spec:covers WCAG22-LABELS-ERRORS: サーバーが拒む JWKS URL を欄のエラー文で示し、送信を止めること
  it('rejects a JWKS URL that is not https or has a fragment', async () => {
    const fetchMock = mock(() => Promise.resolve(response(201, {})))
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" resourceServers={[server]} />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), { target: { value: 'docs-bot' } })
    fireEvent.click(screen.getByLabelText(flow.authJwksUri))
    const jwks = screen.getByLabelText(flow.jwksUriLabel)
    for (const value of ['http://keys.example/jwks.json', 'https://keys.example/jwks.json#k']) {
      fireEvent.change(jwks, { target: { value } })
      expect(jwks.getAttribute('aria-invalid')).toBe('true')
      expect(jwks).toHaveAccessibleDescription(
        'Enter a URL that starts with https:// and has no user name or # fragment.',
      )
    }
    const submit = screen.getByRole('button', { name: t.register })
    expect(submit).toBeDisabled()
    fireEvent.submit(submit.closest('form') as HTMLFormElement)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('retries only the unfinished steps after a partial failure', async () => {
    const calls = routedFetch(1)
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" resourceServers={[server]} />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), { target: { value: 'docs-bot' } })
    fireEvent.change(screen.getByLabelText(flow.publicKeyLabel), { target: { value: publicJWK } })
    fireEvent.click(screen.getByRole('button', { name: t.register }))
    expect(await screen.findByText('Binding failed.')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('agent-client')

    fireEvent.click(screen.getByRole('button', { name: t.register }))
    expect(await screen.findByText(flow.resultHeading)).toBeInTheDocument()
    const paths = calls.map((call) => new URL(call.url, 'http://x').pathname)
    expect(paths.filter((path) => path.endsWith('/api/admin/v1/agents'))).toHaveLength(1)
    expect(paths.filter((path) => path.endsWith('/api/admin/v1/clients'))).toHaveLength(1)
    expect(paths.filter((path) => path.endsWith('/credentials'))).toHaveLength(2)
  })
})
