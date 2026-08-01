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

  it('splits comma and whitespace separated scopes when registering', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(
          response(201, {
            resource_server_id: 'resource-server-1',
            resource: 'https://mcp.example.com',
            name: 'Example MCP',
            scopes: ['mcp.read', 'mcp.write', 'profile'],
            state: 'Active',
          }),
        ),
      ),
    )
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.resourceLabel), {
      target: { value: 'https://mcp.example.com' },
    })
    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: 'Example MCP' } })
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
            resource: 'https://mcp.example.com',
            name: 'Example MCP',
            scopes: ['mcp.read', 'mcp.write', 'profile'],
            state: 'Active',
          }),
        }),
      ),
    )
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/mcp-resource-servers'),
    )
  })

  it('shows an error and keeps the form when registration fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(response(409, { message: 'This resource is already registered.' })),
      ),
    )
    await renderWithRouter(<AdminMcpResourceServerCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.resourceLabel), {
      target: { value: 'https://mcp.example.com' },
    })
    fireEvent.change(screen.getByLabelText(t.nameLabel), { target: { value: 'Example MCP' } })
    fireEvent.change(screen.getByLabelText(t.scopesLabel), { target: { value: 'mcp.read' } })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText('This resource is already registered.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
