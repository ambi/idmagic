import { createFileRoute } from '@tanstack/react-router'
import { getMcpResourceServer, listAdminAgents, listRelationTuples } from '../../../api/admin'
import { AdminMcpAgentAccessPage } from '../../../features/admin-mcp-resource-servers/AdminMcpAgentAccessPage'
import { MCP_OPERATION_TYPE } from '../../../lib/mcpAccess'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/mcp-resource-servers_/$resourceServerId/access')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [server, agents, result] = await Promise.all([
      getMcpResourceServer(params.resourceServerId),
      listAdminAgents(),
      listRelationTuples({ resource_type: MCP_OPERATION_TYPE, subject_type: 'agent' }),
    ])
    return { actorUsername: account.preferred_username, server, agents, tuples: result.tuples }
  },
  component: () => (
    <PageMarker kind="admin-mcp-agent-access">
      <AdminMcpAgentAccessPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
