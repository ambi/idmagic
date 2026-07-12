import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAgentsPage } from './AdminAgentsPage'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import type { AdminAgent } from '../../types'

const t = adminAgentsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
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

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the agent list in English by default', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { agents: [] }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[]} />)
    expect(
      screen.getByRole('heading', { name: adminAgentsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminAgentsDictionary.en.selectAgentPrompt)).toBeInTheDocument()
  })

  it('renders the agent list in Japanese when explicitly selected', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { agents: [] }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[]} />, { locale: 'ja' })
    expect(
      screen.getByRole('heading', { name: adminAgentsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminAgentsPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('registers an agent and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/agents') && init?.method === 'POST') {
          return Promise.resolve(response(201, { ...agent, id: 'agent-2', name: 'billing-bot' }))
        }
        if (url.includes('/api/admin/agents')) {
          return Promise.resolve(response(200, { agents: [agent] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: t.register }))
    const nameInput = await screen.findByLabelText(t.agentNameLabel)
    fireEvent.change(nameInput, { target: { value: 'billing-bot' } })
    const form = nameInput.closest('form') as HTMLFormElement
    fireEvent.click(within(form).getByRole('button', { name: t.register }))

    expect(await screen.findByText(t.agentRegisteredNotice)).toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when registration fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'This agent name is already in use.' })),
      ),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: t.register }))
    const nameInput = await screen.findByLabelText(t.agentNameLabel)
    fireEvent.change(nameInput, { target: { value: 'billing-bot' } })
    const form = nameInput.closest('form') as HTMLFormElement
    fireEvent.click(within(form).getByRole('button', { name: t.register }))

    expect(await screen.findByText('This agent name is already in use.')).toBeInTheDocument()
  })

  it('deletes an agent and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/agents/agent-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/agents')) {
          return Promise.resolve(response(200, { agents: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: t.deleteAgent }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText(t.agentDeletedNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectAgentPrompt)).toBeInTheDocument()
  })

  it('shows an error when binding a credential fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(409, { message: 'This client_id is already in use.' }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.change(screen.getByLabelText(t.bindClientIdAria), {
      target: { value: 'client-x' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.bind }))

    expect(await screen.findByText('This client_id is already in use.')).toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when editing an agent fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: 'Could not update the agent.' }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(await screen.findByLabelText(t.agentNameLabel), {
      target: { value: 'renamed-bot' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the agent.')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
