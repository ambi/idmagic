import { createFileRoute } from '@tanstack/react-router'
import { listMcpResourceServers } from '../../../api/admin'
import { AuthenticationAPIError } from '../../../api/core'
import { AdminMcpResourceServerEditPage } from '../../../features/admin-mcp-resource-servers/AdminMcpResourceServerEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/mcp-resource-servers_/$resourceServerId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const resourceServer = (await listMcpResourceServers()).find(
      (item) => item.id === params.resourceServerId,
    )
    if (!resourceServer) {
      throw new AuthenticationAPIError('MCP resource server not found.', 'not_found')
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      resourceServer,
    }
  },
  component: AdminMcpResourceServerEditRoute,
})

function AdminMcpResourceServerEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-mcp-resource-server-edit">
      <AdminMcpResourceServerEditPage {...data} />
    </PageMarker>
  )
}
