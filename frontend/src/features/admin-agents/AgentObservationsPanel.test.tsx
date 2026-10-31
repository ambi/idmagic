import { afterEach, describe, expect, it, mock, spyOn } from 'bun:test'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import * as api from '../../api'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAgent, AgentObservation } from '../../types'
import { AgentObservationsPanel } from './AgentObservationsPanel'
import { agentObservationsDictionary } from './AgentObservationsPanel.i18n'

const t = agentObservationsDictionary.en

const agent = (id: string): AdminAgent => ({
  id,
  tenant_id: 'tenant-1',
  name: id,
  kind: 'autonomous',
  owner_user_id: 'user-1',
  status: 'active',
  roles: [],
  client_ids: [],
  created_at: '2026-01-01T00:00:00Z',
})

const observation = (overrides: Partial<AgentObservation>): AgentObservation => ({
  id: 'observation-1',
  issuer: 'https://discovery.example',
  subject: 'svc-1',
  resources: [],
  observed_at: '2026-10-01T00:00:00Z',
  source_token_id: 'token-1',
  state: 'unregistered',
  stale: false,
  candidates: [],
  ...overrides,
})

afterEach(() => mock.restore())

describe('AgentObservationsPanel', () => {
  it('distinguishes unverified observations, shows source, freshness, and evidence', async () => {
    await renderWithRouter(
      <AgentObservationsPanel
        csrfToken="csrf"
        agents={[agent('bot-a'), agent('bot-b')]}
        observations={[
          observation({
            state: 'candidate',
            stale: true,
            responsible_candidate: 'someone@example.com',
            candidates: [
              { agent_id: 'bot-a', agent_name: 'bot-a', reasons: ['workload_binding'] },
              { agent_id: 'bot-b', agent_name: 'bot-b', reasons: ['client_binding'] },
            ],
          }),
        ]}
      />,
    )
    const row = within(screen.getByRole('row', { name: /svc-1/ }))
    expect(row.getByText('token-1')).toBeTruthy()
    expect(row.getByText(t.staleLabel)).toBeTruthy()
    expect(row.getByText(t.stateCandidate)).toBeTruthy()
    expect(row.getByText(t.ambiguousLabel)).toBeTruthy()
    expect(row.getByText(`bot-a: ${t.reasonWorkloadBinding}`)).toBeTruthy()
    expect(row.getByText(/someone@example.com/)).toBeTruthy()
  })

  it('requires choosing an agent when no candidate exists, then confirms the chosen agent', async () => {
    const confirm = spyOn(api, 'confirmAgentObservation').mockResolvedValue(observation({}))
    spyOn(api, 'listAgentObservations').mockResolvedValue([])
    await renderWithRouter(
      <AgentObservationsPanel
        csrfToken="csrf"
        agents={[agent('bot-a')]}
        observations={[observation({})]}
      />,
    )
    const button = screen.getByRole('button', { name: t.confirmAction }) as HTMLButtonElement
    expect(button.disabled).toBe(true)
    fireEvent.change(screen.getByLabelText(t.agentSelectLabel), { target: { value: 'bot-a' } })
    fireEvent.click(button)
    await waitFor(() => expect(confirm).toHaveBeenCalledWith('csrf', 'observation-1', 'bot-a'))
  })

  it('shows the refusal when another administrator already decided', async () => {
    spyOn(api, 'excludeAgentObservation').mockRejectedValue(new Error('already decided'))
    await renderWithRouter(
      <AgentObservationsPanel csrfToken="csrf" agents={[]} observations={[observation({})]} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.excludeAction }))
    expect((await screen.findByRole('alert')).textContent).toContain('already decided')
  })

  it('offers only returning to pending for a decided observation', async () => {
    await renderWithRouter(
      <AgentObservationsPanel
        csrfToken="csrf"
        agents={[agent('bot-a')]}
        observations={[observation({ state: 'confirmed', confirmed_agent_id: 'bot-a' })]}
      />,
    )
    expect(screen.getByRole('button', { name: t.reopenAction })).toBeTruthy()
    expect(screen.queryByRole('button', { name: t.confirmAction })).toBeNull()
  })
})
