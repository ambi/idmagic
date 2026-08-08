import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { AdminAgentEditPage } from './AdminAgentEditPage'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import type { AdminAgent } from '../../types'

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

describe('AdminAgentEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('updates the agent and redirects to its detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { ...agent, name: 'renamed-bot' }))),
    )
    await renderWithRouter(<AdminAgentEditPage csrfToken="csrf" agent={agent} />)

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

  it('shows an error when binding a credential fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'This client_id is already in use.' }))),
    )
    await renderWithRouter(<AdminAgentEditPage csrfToken="csrf" agent={agent} />)

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
      <AdminAgentEditPage csrfToken="csrf" agent={{ ...agent, client_ids: ['client-a'] }} />,
    )

    expect(screen.getByText('client-a')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.unbind }))

    await waitFor(() => expect(screen.getByText(t.noCredentialsNotice)).toBeInTheDocument())
  })
})
