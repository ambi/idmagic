import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { McpResourceServer } from '../../types'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { AdminMcpResourceServersPage } from './AdminMcpResourceServersPage'

const resourceServer: McpResourceServer = {
  tenant_id: 'tenant-1',
  resource_server_id: 'resource-server-1',
  resource: 'https://mcp.example.com',
  name: 'Example MCP',
  scopes: ['mcp.read', 'mcp.write'],
  state: 'Active',
  created_at: '2026-07-20T00:00:00Z',
  updated_at: '2026-07-20T00:00:00Z',
}

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

describe('AdminMcpResourceServersPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the resource and scopes in English by default', async () => {
    await renderWithRouterBase(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={[resourceServer]} />,
    )
    expect(
      screen.getByRole('heading', { name: adminMcpResourceServersDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: adminMcpResourceServersDictionary.en.registerResourceServer,
      }),
    ).toBeInTheDocument()
    expect(screen.getByText(resourceServer.resource)).toBeInTheDocument()
    expect(screen.getByText(resourceServer.scopes[0])).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={[resourceServer]} />,
    )
    expect(
      screen.getByRole('heading', { name: adminMcpResourceServersDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: adminMcpResourceServersDictionary.ja.registerResourceServer,
      }),
    ).toBeInTheDocument()
  })

  it('shows an empty state when no resource servers are registered', async () => {
    await renderWithRouterBase(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={[]} />,
    )
    expect(screen.getByText(adminMcpResourceServersDictionary.en.emptyNotice)).toBeInTheDocument()
  })

  it('splits comma and whitespace separated scopes when registering', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: vi.fn().mockResolvedValue(resourceServer),
      }),
    )
    const t = adminMcpResourceServersDictionary.en
    await renderWithRouterBase(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={[]} />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.registerResourceServer }))
    fireEvent.change(screen.getByLabelText(t.resourceLabel), {
      target: { value: resourceServer.resource },
    })
    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: resourceServer.name } })
    fireEvent.change(screen.getByLabelText(t.scopesLabel), {
      target: { value: 'mcp.read,  mcp.write profile' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/admin/mcp-resource-servers'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            resource: resourceServer.resource,
            name: resourceServer.name,
            scopes: ['mcp.read', 'mcp.write', 'profile'],
            state: 'Active',
          }),
        }),
      ),
    )
  })
})
