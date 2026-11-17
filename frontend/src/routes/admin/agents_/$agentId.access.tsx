import { createFileRoute } from '@tanstack/react-router'
import {
  getAgentAccessSummary,
  getAgentGuardrailPolicy,
  getMcpToolCatalog,
} from '../../../api/admin'
import { AdminAgentAccessPage } from '../../../features/admin-agents/AdminAgentAccessPage'
import { PageMarker } from '../../-page'
import { loadAgentResources } from './-resources'

export const Route = createFileRoute('/admin/agents_/$agentId/access')({
  loader: async ({ location, params }) => {
    const resources = await loadAgentResources(
      location.pathname,
      location.searchStr,
      params.agentId,
    )
    const [summary, guardrails, catalogEntries] = await Promise.all([
      getAgentAccessSummary(params.agentId),
      getAgentGuardrailPolicy(params.agentId),
      Promise.all(
        resources.resourceServers.map(async (server) => {
          try {
            return [server.id, await getMcpToolCatalog(server.id)] as const
          } catch {
            return [server.id, 'failed' as const]
          }
        }),
      ),
    ])
    return { ...resources, summary, guardrails, catalogs: Object.fromEntries(catalogEntries) }
  },
  component: () => (
    <PageMarker kind="admin-agent-access">
      <AdminAgentAccessPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
