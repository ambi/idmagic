import { screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
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

  it('links "register resource server" to the dedicated create route instead of an inline form', async () => {
    await renderWithRouterBase(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={[resourceServer]} />,
    )
    expect(
      screen.getByRole('button', {
        name: adminMcpResourceServersDictionary.en.registerResourceServer,
      }),
    ).toHaveAttribute('href', '/admin/mcp-resource-servers/new')
    expect(
      screen.getByRole('button', { name: adminMcpResourceServersDictionary.en.edit }),
    ).toHaveAttribute('href', '/admin/mcp-resource-servers/resource-server-1/edit')
  })
})
