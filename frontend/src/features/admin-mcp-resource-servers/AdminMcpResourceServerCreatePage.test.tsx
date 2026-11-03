import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { AdminMcpResourceServerCreatePage } from './AdminMcpResourceServerCreatePage'

const t = adminMcpResourceServersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AdminMcpResourceServerCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it.each(['[null]', '[{"scopes":{}}]', '[{"resource_type":{}}]'])(
    'keeps invalid rule JSON editable: %s',
    async (value) => {
      await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)
      fireEvent.click(screen.getByText(t.scopesAdvanced))
      fireEvent.change(screen.getByLabelText(t.scopesLabel), { target: { value: 'tools:read' } })

      fireEvent.change(screen.getByLabelText(t.accessRulesLabel), { target: { value } })

      expect(screen.getByLabelText(t.accessRulesLabel)).toHaveValue(value)
      expect(screen.getByText(t.accessRulesUnreadable)).toBeInTheDocument()
      fireEvent.change(screen.getByLabelText(t.accessRulesLabel), { target: { value: '[]' } })
      expect(screen.getByRole('button', { name: t.ruleAdd })).toBeInTheDocument()
    },
  )

  function openScopes() {
    fireEvent.click(screen.getByText(t.scopesAdvanced))
  }

  function fill(name = 'Example MCP') {
    fireEvent.change(screen.getByLabelText(t.resourceLabel), {
      target: { value: 'https://mcp.example.com' },
    })
    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: name } })
  }

  // routedFetch は URL と方法ごとに応答を返し、呼び出しを記録する。
  function routedFetch(serverResponses: ReturnType<typeof response>[]) {
    const calls: { url: string; method: string; body?: unknown }[] = []
    const fetchMock = mock((url: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET'
      calls.push({ url, method, body: init?.body ? JSON.parse(String(init.body)) : undefined })
      if (url.includes('/api/admin/v1/applications') && method === 'GET') {
        return Promise.resolve(
          response(200, {
            applications: [
              {
                id: 'app-1',
                name: 'Existing gateway',
                kind: 'service',
                protocol: { type: 'oidc', client_id: 'existing-client' },
              },
              {
                id: 'app-2',
                name: 'Web app',
                kind: 'federated',
                protocol: { type: 'oidc', client_id: 'web-client' },
              },
            ],
          }),
        )
      }
      if (url.includes('/api/admin/v1/applications') && method === 'POST') {
        return Promise.resolve(
          response(201, {
            application: { id: 'app-3' },
            client_id: 'new-client',
            client_secret: 'new-secret',
          }),
        )
      }
      return Promise.resolve(serverResponses.shift() ?? response(500))
    })
    stubGlobal('fetch', fetchMock)
    return calls
  }

  const created = response(201, {
    id: 'resource-server-1',
    resource: 'https://mcp.example.com',
    name: 'Example MCP',
    scopes: ['mcp'],
    state: 'Active',
  })

  it('splits comma and whitespace separated scopes when registering', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const calls = routedFetch([created])
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fill()
    fireEvent.change(screen.getByLabelText(t.evaluatorLabel), { target: { value: '' } })
    openScopes()
    fireEvent.change(screen.getByLabelText(t.scopesLabel), {
      target: { value: 'mcp.read,  mcp.write profile' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/mcp-resource-servers'),
    )
    expect(calls.find((call) => call.url.includes('mcp-resource-servers'))?.body).toEqual({
      resource: 'https://mcp.example.com',
      name: 'Example MCP',
      scopes: ['mcp.read', 'mcp.write', 'profile'],
      state: 'Active',
    })
  })

  //spec:covers REQ-OAUTH2-088: 評価用のクライアントを同じ送信で作成して束縛し、シークレットと Gateway の設定を示す。client_id とスコープは入力しない。
  it('creates and binds a new evaluation client and shows the gateway configuration', async () => {
    const calls = routedFetch([created])
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fill()
    expect(
      screen.getByText(/create and bind a new evaluation client "Example MCP Gateway"/),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText(t.resultHeading)).toBeInTheDocument()
    const application = calls.find(
      (call) => call.url.includes('/applications') && call.method === 'POST',
    )
    expect(application?.body).toEqual({ name: 'Example MCP Gateway', type: 'service' })
    expect(calls.find((call) => call.url.includes('mcp-resource-servers'))?.body).toEqual({
      resource: 'https://mcp.example.com',
      name: 'Example MCP',
      scopes: ['mcp'],
      client_id: 'new-client',
      state: 'Active',
    })
    expect(document.querySelector('#result-client-secret')).toHaveTextContent('new-secret')
    const config = document.querySelector('#result-gateway-config')?.textContent ?? ''
    expect(config).toContain('resource: https://mcp.example.com')
    expect(config).toContain('client_id: "new-client"')
    expect(config).toContain('catalog: true')
    expect(config).not.toContain('new-secret')
  })

  //spec:covers REQ-OAUTH2-088: 既存のサービスは名前で選び、client_id を入力しない。
  it('binds an existing service chosen by name', async () => {
    const calls = routedFetch([created])
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fill()
    await screen.findByRole('option', { name: 'Existing gateway' })
    expect(screen.queryByRole('option', { name: 'Web app' })).toBeNull()
    fireEvent.change(screen.getByLabelText(t.evaluatorLabel), {
      target: { value: 'existing-client' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText(t.resultHeading)).toBeInTheDocument()
    expect(calls.some((call) => call.url.includes('/applications') && call.method === 'POST')).toBe(
      false,
    )
    expect(document.querySelector('#result-client-id')).toHaveTextContent('existing-client')
    expect(document.querySelector('#result-client-secret')).toBeNull()
  })

  //spec:covers REQ-OAUTH2-088: 登録だけが失敗したら作成済みのクライアントを保ち、再試行で作り直さない。
  it('keeps the created client and retries only the registration', async () => {
    const calls = routedFetch([
      response(409, { message: 'This resource is already registered.' }),
      created,
    ])
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fill()
    fireEvent.click(screen.getByRole('button', { name: t.register }))
    expect(await screen.findByText('This resource is already registered.')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('new-client')

    fireEvent.click(screen.getByRole('button', { name: t.register }))
    expect(await screen.findByText(t.resultHeading)).toBeInTheDocument()
    expect(
      calls.filter((call) => call.url.includes('/applications') && call.method === 'POST'),
    ).toHaveLength(1)
    expect(document.querySelector('#result-client-secret')).toHaveTextContent('new-secret')
  })

  //spec:covers REQ-OAUTH2-064, REQ-OAUTH2-088: スコープが一つなら行ごとの選択を出さず、ツールの行はそのスコープを使う。
  it('hides per-tool scope choices while only the default scope exists', async () => {
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fireEvent.click(screen.getByRole('button', { name: t.ruleAdd }))
    expect(screen.getByText(t.ruleScopes).closest('fieldset')).not.toBeVisible()
    expect(
      JSON.parse((screen.getByLabelText(t.accessRulesLabel) as HTMLTextAreaElement).value)[0]
        .scopes,
    ).toEqual(['mcp'])
  })
})
