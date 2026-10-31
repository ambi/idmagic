import { describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAgent, AgentInventory } from '../../types'
import { AdminAgentInventoryPage } from './AdminAgentInventoryPage'
import { adminAgentInventoryDictionary } from './AdminAgentInventoryPage.i18n'

const t = adminAgentInventoryDictionary.en

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

const inventory: AgentInventory = {
  dormant_days: 90,
  agents: [
    {
      agent: agent('healthy-bot'),
      owner_status: 'active',
      last_token_issued_at: '2026-10-01T00:00:00Z',
      missing_client_ids: [],
      findings: [],
      confirmed_observations: [],
    },
    {
      agent: agent('ghost-bot'),
      owner_status: 'disabled',
      missing_client_ids: ['lost-client'],
      findings: ['owner_missing', 'unbound'],
      confirmed_observations: [],
    },
  ],
}

describe('AdminAgentInventoryPage', () => {
  it('shows findings, owner status, and missing clients for each agent', async () => {
    await renderWithRouter(
      <AdminAgentInventoryPage
        csrfToken="csrf"
        observations={[]}
        inventory={inventory}
        onFilter={mock()}
      />,
    )

    const ghost = screen.getByRole('row', { name: /ghost-bot/ })
    expect(ghost).toHaveTextContent(t.findingOwnerMissing)
    expect(ghost).toHaveTextContent(t.findingUnbound)
    expect(ghost).toHaveTextContent(t.ownerDisabled)
    expect(ghost).toHaveTextContent('lost-client')
    expect(ghost).toHaveTextContent(t.neverIssued)
    expect(screen.getByRole('row', { name: /healthy-bot/ })).toHaveTextContent(t.noFindings)
  })

  it('filters by finding and keeps the current dormant days', async () => {
    const onFilter = mock()
    await renderWithRouter(
      <AdminAgentInventoryPage
        csrfToken="csrf"
        observations={[]}
        inventory={inventory}
        onFilter={onFilter}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.findingDormant }))

    expect(onFilter).toHaveBeenCalledWith({ finding: 'dormant', dormantDays: 90 })
  })

  it('applies the entered dormant days and keeps the current finding', async () => {
    const onFilter = mock()
    await renderWithRouter(
      <AdminAgentInventoryPage
        csrfToken="csrf"
        observations={[]}
        inventory={inventory}
        finding="unbound"
        onFilter={onFilter}
      />,
    )

    fireEvent.change(screen.getByLabelText(t.dormantDaysLabel), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: t.apply }))

    expect(onFilter).toHaveBeenCalledWith({ finding: 'unbound', dormantDays: 30 })
  })

  it('tells the administrator when no agent matches', async () => {
    await renderWithRouter(
      <AdminAgentInventoryPage
        csrfToken="csrf"
        observations={[]}
        inventory={{ dormant_days: 90, agents: [] }}
        onFilter={mock()}
      />,
    )

    expect(screen.getByText(t.emptyNotice)).toBeInTheDocument()
  })
})
