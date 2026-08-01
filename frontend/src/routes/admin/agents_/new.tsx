import { createFileRoute } from '@tanstack/react-router'
import { AdminAgentCreatePage } from '../../../features/admin-agents/AdminAgentCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/agents_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminAgentCreateRoute,
})

function AdminAgentCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-agent-create">
      <AdminAgentCreatePage {...data} />
    </PageMarker>
  )
}
