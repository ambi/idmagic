import { createFileRoute } from '@tanstack/react-router'
import { listAdminAgentsPage } from '../../api'
import { AdminAgentsPage } from '../../features/admin-agents/AdminAgentsListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/agents')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const page = await listAdminAgentsPage()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      agents: page.agents,
      nextCursor: page.nextCursor,
    }
  },
  component: AdminAgentsRoute,
})

function AdminAgentsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-agents">
      <AdminAgentsPage {...data} />
    </PageMarker>
  )
}
