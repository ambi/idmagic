import { createFileRoute } from '@tanstack/react-router'
import { listMcpResourceServers } from '../../api/admin'
import { AdminMcpResourceServersPage } from '../../features/admin-mcp-resource-servers/AdminMcpResourceServersPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/mcp-resource-servers')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const resourceServers = await listMcpResourceServers()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      resourceServers,
    }
  },
  component: AdminMcpResourceServersRoute,
})

function AdminMcpResourceServersRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-mcp-resource-servers">
      <AdminMcpResourceServersPage {...data} />
    </PageMarker>
  )
}
