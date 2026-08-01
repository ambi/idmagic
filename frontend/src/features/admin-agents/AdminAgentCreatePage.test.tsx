import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { AdminAgentCreatePage } from './AdminAgentCreatePage'
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

describe('AdminAgentCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('registers an agent and redirects to its detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(201, { ...agent, id: 'agent-2' }))),
    )
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), {
      target: { value: 'billing-bot' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/agents/agent-2'),
    )
  })

  it('shows an error and keeps the form when registration fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'This agent name is already in use.' }))),
    )
    await renderWithRouter(<AdminAgentCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.agentNameLabel), {
      target: { value: 'billing-bot' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText('This agent name is already in use.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
