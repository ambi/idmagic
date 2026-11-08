import { fireEvent, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import type { McpResourceServer } from '../../types'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { AdminMcpResourceServersPage } from './AdminMcpResourceServersPage'

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

const servers: McpResourceServer[] = [
  {
    tenant_id: 'default',
    created_at: '',
    updated_at: '',
    id: 'one',
    name: 'Documents',
    resource: 'https://docs.example/mcp',
    state: 'Active',
    scopes: ['docs'],
    client_id: 'docs-client',
    access_rules: [
      {
        scopes: ['docs'],
        resource_type: 'idmagic_mcp_operation',
        relation: 'invoke',
        tool: { name: 'read', display_name: 'Read' },
      },
    ],
  },
  {
    tenant_id: 'default',
    created_at: '',
    updated_at: '',
    id: 'two',
    name: 'Archive',
    resource: 'https://archive.example/mcp',
    state: 'Disabled',
    scopes: ['archive'],
    client_id: 'archive-client',
  },
]

describe('AdminMcpResourceServersPage', () => {
  it('selects a server in the table and shows its evaluation client and actions', async () => {
    await renderWithRouterBase(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={servers} />,
    )
    const table = screen.getByRole('table')
    const row = within(table).getByRole('row', { name: /Documents/ })
    expect(row).toHaveTextContent('Active')
    expect(row).toHaveTextContent('1')
    expect(screen.getByText('docs-client')).toBeInTheDocument()
    fireEvent.click(within(table).getByRole('button', { name: 'Show details for Archive' }))
    expect(screen.getByText('archive-client')).toBeInTheDocument()
    expect(screen.queryByText('docs-client')).toBeNull()
    expect(screen.getByRole('button', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/admin/mcp-resource-servers/two/edit',
    )
    fireEvent.click(screen.getByRole('button', { name: '日本語' }))
    expect(within(table).getByText('有効')).toBeInTheDocument()
    expect(within(table).getByText('無効')).toBeInTheDocument()
  })

  it('explains how to add the first server', async () => {
    await renderWithRouterBase(
      <AdminMcpResourceServersPage csrfToken="csrf" resourceServers={[]} />,
    )
    expect(screen.getByText('Register one using Add resource server.')).toBeInTheDocument()
  })
})
