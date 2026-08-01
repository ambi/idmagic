import { createFileRoute } from '@tanstack/react-router'
import { AdminMcpResourceServerCreatePage } from '../../../features/admin-mcp-resource-servers/AdminMcpResourceServerCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/mcp-resource-servers_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminMcpResourceServerCreateRoute,
})

function AdminMcpResourceServerCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-mcp-resource-server-create">
      <AdminMcpResourceServerCreatePage {...data} />
    </PageMarker>
  )
}
