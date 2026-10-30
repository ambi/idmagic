import { createFileRoute } from '@tanstack/react-router'
import { getAgentInventory } from '../../../api'
import { AdminAgentInventoryPage } from '../../../features/admin-agents/AdminAgentInventoryPage'
import type { AgentInventoryFinding } from '../../../types'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

const FINDINGS: readonly string[] = ['owner_missing', 'dormant', 'unbound']

export const Route = createFileRoute('/admin/agents_/inventory')({
  validateSearch: (search: Record<string, unknown>) => ({
    ...(typeof search.finding === 'string' && FINDINGS.includes(search.finding)
      ? { finding: search.finding as AgentInventoryFinding }
      : {}),
    ...(typeof search.dormant_days === 'number' && Number.isInteger(search.dormant_days)
      ? { dormant_days: search.dormant_days }
      : {}),
  }),
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const inventory = await getAgentInventory({
      finding: deps.finding,
      dormantDays: deps.dormant_days,
    })
    return { actorUsername: account.preferred_username, inventory }
  },
  component: AdminAgentInventoryRoute,
})

function AdminAgentInventoryRoute() {
  const data = Route.useLoaderData()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <PageMarker kind="admin-agent-inventory">
      <AdminAgentInventoryPage
        key={`${search.finding ?? ''}:${data.inventory.dormant_days}`}
        actorUsername={data.actorUsername}
        inventory={data.inventory}
        finding={search.finding}
        onFilter={({ finding, dormantDays }) =>
          navigate({
            search: {
              ...(finding ? { finding } : {}),
              ...(dormantDays ? { dormant_days: dormantDays } : {}),
            },
          })
        }
      />
    </PageMarker>
  )
}
