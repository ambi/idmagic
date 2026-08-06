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
      mock(() => Promise.resolve(response(200, { ...resourceServer, name: 'Renamed MCP' }))),
    )
    await renderWithRouter(
      <AdminMcpResourceServerEditPage csrfToken="csrf" resourceServer={resourceServer} />,
    )

    expect(screen.getByLabelText(t.resourceLabel)).toBeDisabled()

    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: 'Renamed MCP' } })
    fireEvent.click(screen.getByRole('button', { name: t.update }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/admin/mcp-resource-servers/resource-server-1'),
        expect.objectContaining({ method: 'PATCH' }),
      ),
    )
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/mcp-resource-servers'),
    )
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
