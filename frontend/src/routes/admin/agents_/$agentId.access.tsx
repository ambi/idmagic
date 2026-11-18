import { createFileRoute } from '@tanstack/react-router'
import {
  getAgentAccessSummary,
  getAgentGuardrailPolicy,
  getMcpToolCatalog,
} from '../../../api/admin'
import { StepUpRequiredError, UnauthenticatedError } from '../../../api/core'
import { AdminAgentAccessPage } from '../../../features/admin-agents/AdminAgentAccessPage'
import type { CatalogResult } from '../../../features/admin-agents/mcpConnectionCheck'
import type { McpResourceServer } from '../../../types'
import { PageMarker } from '../../-page'
import { loadAgentResources } from './-resources'

// loadToolCatalogs は接続先ごとのカタログを集める。一つの取得の失敗は画面全体を止めず、
// その接続先に 'failed' を置く。再認証を要する失敗は、ルーターに扱わせるため投げ直す。
async function loadToolCatalogs(
  servers: McpResourceServer[],
): Promise<Record<string, CatalogResult>> {
  const entries = await Promise.all(
    servers.map(async (server): Promise<[string, CatalogResult]> => {
      try {
        return [server.id, await getMcpToolCatalog(server.id)]
      } catch (error) {
        if (error instanceof UnauthenticatedError || error instanceof StepUpRequiredError) {
          throw error
        }
        return [server.id, 'failed']
      }
    }),
  )
  return Object.fromEntries(entries)
}

export const Route = createFileRoute('/admin/agents_/$agentId/access')({
  loader: async ({ location, params }) => {
    const resources = await loadAgentResources(
      location.pathname,
      location.searchStr,
      params.agentId,
    )
    const [summary, guardrails, catalogs] = await Promise.all([
      getAgentAccessSummary(params.agentId),
      getAgentGuardrailPolicy(params.agentId),
      loadToolCatalogs(resources.resourceServers),
    ])
    return { ...resources, summary, guardrails, catalogs }
  },
  component: () => (
    <PageMarker kind="admin-agent-access">
      <AdminAgentAccessPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
