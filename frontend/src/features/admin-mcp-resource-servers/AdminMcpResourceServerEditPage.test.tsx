import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { AdminMcpResourceServerEditPage } from './AdminMcpResourceServerEditPage'

const t = adminMcpResourceServersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const resourceServer: McpResourceServer = {
  tenant_id: 'tenant-1',
  id: 'resource-server-1',
  resource: 'https://mcp.example.com',
  name: 'Example MCP',
  scopes: ['mcp.read', 'mcp.write'],
  state: 'Active',
  created_at: '2026-07-20T00:00:00Z',
  updated_at: '2026-07-20T00:00:00Z',
}

describe('AdminMcpResourceServerEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('locks the resource URI field and updates the resource server', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock((url: string) =>
        Promise.resolve(
          url.includes('/api/admin/v1/mcp-resource-servers/resource-server-1') &&
            !url.includes('reported')
            ? response(200, { ...resourceServer, name: 'Renamed MCP' })
            : response(200, { tools: [], complete: false, applications: [] }),
        ),
      ),
    )
    await renderWithRouter(
      <AdminMcpResourceServerEditPage csrfToken="csrf" resourceServer={resourceServer} />,
    )

    expect(screen.getByLabelText(t.resourceLabel)).toBeDisabled()

    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: 'Renamed MCP' } })
    fireEvent.click(screen.getByRole('button', { name: t.update }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/admin/v1/mcp-resource-servers/resource-server-1'),
        expect.objectContaining({ method: 'PATCH' }),
      ),
    )
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/mcp-resource-servers'),
    )
  })

  // routedFetch は一覧と報告の読み込みに応答し、保存の要求を記録する。
  function routedFetch(
    save: ReturnType<typeof response>,
    reported: unknown = { tools: [], complete: false },
  ) {
    const saved: unknown[] = []
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.includes('/reported-tools')) return Promise.resolve(response(200, reported))
        if (url.includes('/api/admin/v1/applications')) {
          return Promise.resolve(
            response(200, {
              applications: [
                {
                  id: 'app-1',
                  name: 'Docs gateway',
                  kind: 'service',
                  protocol: { type: 'oidc', client_id: 'rs-client' },
                },
              ],
            }),
          )
        }
        saved.push(JSON.parse(String(init?.body)))
        return Promise.resolve(save)
      }),
    )
    return saved
  }

  it('sends the evaluation client and rules and refuses rules that are not a JSON array', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const saved = routedFetch(response(200, resourceServer))
    await renderWithRouter(
      <AdminMcpResourceServerEditPage csrfToken="csrf" resourceServer={resourceServer} />,
    )

    await screen.findByRole('option', { name: 'Docs gateway' })
    expect(screen.queryByRole('option', { name: t.evaluatorNew })).toBeNull()
    fireEvent.change(screen.getByLabelText(t.evaluatorLabel), { target: { value: 'rs-client' } })
    fireEvent.change(screen.getByLabelText(t.accessRulesLabel), { target: { value: '{}' } })
    fireEvent.click(screen.getByRole('button', { name: t.update }))
    expect(await screen.findByText(t.accessRulesInvalidError)).toBeInTheDocument()
    expect(saved).toHaveLength(0)

    const rules = [{ resource_type: 'document', relation: 'viewer', scopes: ['mcp.read'] }]
    fireEvent.change(screen.getByLabelText(t.accessRulesLabel), {
      target: { value: JSON.stringify(rules) },
    })
    fireEvent.click(screen.getByRole('button', { name: t.update }))
    await waitFor(() =>
      expect(saved).toEqual([
        {
          name: resourceServer.name,
          scopes: resourceServer.scopes,
          client_id: 'rs-client',
          access_rules: rules,
          state: 'Active',
        },
      ]),
    )
  })

  //spec:covers REQ-OAUTH2-087: 報告されたツールは選ぶまで公開されず、選ぶと名前と表示名を入れた行になる。完全な報告にない公開中のツールは削除せずに示す。
  it('publishes a reported tool only when chosen and flags published tools missing from a complete report', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const published = {
      ...resourceServer,
      scopes: ['mcp'],
      access_rules: [
        {
          resource_type: 'idmagic_mcp_operation',
          relation: 'invoke',
          scopes: ['mcp'],
          tool: { id: 'op-1', name: 'archive_document', display_name: 'Archive' },
        },
      ],
    }
    const saved = routedFetch(response(200, published), {
      complete: true,
      reported_at: '2026-11-03T09:00:00Z',
      tools: [
        {
          name: 'read_document',
          title: 'Read documents',
          description: '<b>reads</b>',
          arguments: ['document_id'],
        },
        { name: 'delete_document' },
      ],
    })
    await renderWithRouter(
      <AdminMcpResourceServerEditPage csrfToken="csrf" resourceServer={published} />,
    )

    expect(await screen.findByText('Read documents')).toBeInTheDocument()
    expect(screen.getByText('<b>reads</b>')).toBeInTheDocument()
    expect(
      screen.getByText(t.reportedMissing.replace('{names}', 'archive_document')),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: `${t.reportedPublish} read_document` }))
    expect(
      screen.getByRole('button', { name: `${t.reportedPublish} delete_document` }),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.update }))

    await waitFor(() => expect(saved).toHaveLength(1))
    const rules = (
      saved[0] as {
        access_rules: { tool?: { name: string; display_name: string }; scopes: string[] }[]
      }
    ).access_rules
    expect(rules.map((rule) => rule.tool?.name)).toEqual(['archive_document', 'read_document'])
    expect(rules[1]).toMatchObject({
      scopes: ['mcp'],
      tool: { name: 'read_document', display_name: 'Read documents' },
    })
  })

  it('shows an error and keeps the form when updating fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Could not update the resource.' }))),
    )
    await renderWithRouter(
      <AdminMcpResourceServerEditPage csrfToken="csrf" resourceServer={resourceServer} />,
    )

    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: 'Renamed MCP' } })
    fireEvent.click(screen.getByRole('button', { name: t.update }))

    expect(await screen.findByText('Could not update the resource.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
