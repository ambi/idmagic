import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import type { AccountAgentDelegation } from '../../types'
import { AgentDelegationsSection } from './AgentDelegationsSection'

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

const delegation: AccountAgentDelegation = {
  agent_id: 'agent-1',
  agent_name: 'Order bot',
  owner_name: 'owner',
  scopes: ['read'],
  state: 'granted',
  granted_at: '2026-01-01T00:00:00Z',
  expires_at: '2027-01-01T00:00:00Z',
}

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AgentDelegationsSection', () => {
  afterEach(() => restoreGlobals())

  it('shows the owner, the scopes, the expiry, and that the grant is unused', async () => {
    await renderWithRouter(
      <AgentDelegationsSection csrfToken="csrf" delegations={[delegation]} delegableAgents={[]} />,
    )
    expect(screen.getByText('Order bot')).toBeInTheDocument()
    expect(screen.getByText(/所有者: owner/)).toBeInTheDocument()
    expect(screen.getByText(/未使用/)).toBeInTheDocument()
    expect(screen.getByText('read')).toBeInTheDocument()
  })

  it('revokes one agent and removes only that agent', async () => {
    const fetchMock = mock().mockResolvedValue(response(204))
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(
      <AgentDelegationsSection
        csrfToken="csrf"
        delegations={[delegation, { ...delegation, agent_id: 'agent-2', agent_name: 'Mail bot' }]}
        delegableAgents={[]}
      />,
    )
    fireEvent.click(screen.getAllByRole('button', { name: /代行を取り消す/ })[0])

    expect(await screen.findByText(/「Order bot」の代行を取り消しました/)).toBeInTheDocument()
    expect(screen.queryByText('Order bot')).toBeNull()
    expect(screen.getByText('Mail bot')).toBeInTheDocument()
    expect(String(fetchMock.mock.calls[0][0])).toContain(
      '/api/account/v1/agent-delegations/agent-1/revoke',
    )
  })

  it('grants the selected agent with the entered scopes', async () => {
    const fetchMock = mock().mockResolvedValue(response(201, delegation))
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(
      <AgentDelegationsSection
        csrfToken="csrf"
        delegations={[]}
        delegableAgents={[{ agent_id: 'agent-1', name: 'Order bot', owner_name: 'owner' }]}
      />,
    )
    fireEvent.change(screen.getByLabelText('エージェント'), { target: { value: 'agent-1' } })
    fireEvent.change(screen.getByLabelText(/許可するスコープ/), {
      target: { value: 'read  write' },
    })
    fireEvent.click(screen.getByRole('button', { name: '代行を許可する' }))

    await waitFor(() => expect(screen.getByText(/に代行を許可しました/)).toBeInTheDocument())
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      agent_id: 'agent-1',
      scopes: ['read', 'write'],
    })
  })

  it('shows the failure and keeps the delegation when revoking fails', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(500)))
    await renderWithRouter(
      <AgentDelegationsSection csrfToken="csrf" delegations={[delegation]} delegableAgents={[]} />,
    )
    fireEvent.click(screen.getByRole('button', { name: /代行を取り消す/ }))
    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
    expect(screen.getByText('Order bot')).toBeInTheDocument()
  })
})
