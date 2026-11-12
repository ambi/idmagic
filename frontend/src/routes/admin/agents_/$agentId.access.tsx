import { createFileRoute } from '@tanstack/react-router'
import { getAgentAccessSummary, getAgentGuardrailPolicy } from '../../../api/admin'
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
    const [summary, guardrails] = await Promise.all([
      getAgentAccessSummary(params.agentId),
      getAgentGuardrailPolicy(params.agentId),
    ])
    return { ...resources, summary, guardrails }
  },
  component: () => (
    <PageMarker kind="admin-agent-access">
      <AdminAgentAccessPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
