import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAgentsPage } from './AdminAgentsListPage'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import type { AdminAgent } from '../../types'

const t = adminAgentsDictionary.en
const tCommon = commonDictionary.en

const response = (status: number, body: unknown = {}, headers: Record<string, string> = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(headers),
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
  afterEach(() => restoreGlobals())

  it('renders the agent list in English by default', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { agents: [] }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[]} nextCursor={null} />)
    expect(
      screen.getByRole('heading', { name: adminAgentsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminAgentsDictionary.en.selectAgentPrompt)).toBeInTheDocument()
  })

  it('renders the agent list in Japanese when explicitly selected', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { agents: [] }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[]} nextCursor={null} />, {
      locale: 'ja',
    })
    expect(
      screen.getByRole('heading', { name: adminAgentsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminAgentsPage', () => {
  afterEach(() => restoreGlobals())

  it('links "add agent" to the dedicated create route instead of a modal', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { agents: [agent] }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} nextCursor={null} />)

    const addLink = screen.getByRole('button', { name: new RegExp(t.addAgent) })
    expect(addLink).toHaveAttribute('href', '/admin/agents/new')
  })

  it('is read-only for credentials: no bind input on the list right pane (wi-314 T012)', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { agents: [agent] }))),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} nextCursor={null} />)

    expect(screen.queryByLabelText(t.bindClientIdAria)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.bind })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: new RegExp(t.edit) })).toHaveAttribute(
      'href',
      '/admin/agents/agent-1/edit',
    )
  })

  it('deletes an agent and refreshes the list on success', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/v1/agents/agent-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/v1/agents')) {
          return Promise.resolve(response(200, { agents: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} nextCursor={null} />)

    fireEvent.click(screen.getByRole('button', { name: t.deleteAgent }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText(t.agentDeletedNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectAgentPrompt)).toBeInTheDocument()
  })

  it('navigates to the next addressable page without appending rows', async () => {
    const onPage = mock()
    await renderWithRouter(
      <AdminAgentsPage csrfToken="csrf" agents={[agent]} nextCursor="abc" onPage={onPage} />,
    )

    fireEvent.click(screen.getByRole('button', { name: tCommon.nextPage }))

    expect(onPage).toHaveBeenCalledWith('abc')
    expect(screen.getAllByText('invoice-bot').length).toBeGreaterThan(0)
  })
})
