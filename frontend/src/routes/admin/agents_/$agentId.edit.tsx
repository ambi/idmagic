import { createFileRoute } from '@tanstack/react-router'
import { getAdminAgent, listAdminUsers } from '../../../api/admin'
import { AdminAgentEditPage } from '../../../features/admin-agents/AdminAgentEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/agents_/$agentId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [agent, users] = await Promise.all([
      getAdminAgent(params.agentId),
      listAdminUsers({ all: true }),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      agent,
      users,
    }
  },
  component: AdminAgentEditRoute,
})

function AdminAgentEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-agent-edit">
      <AdminAgentEditPage {...data} />
    </PageMarker>
  )
}
