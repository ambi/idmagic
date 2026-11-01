import { createFileRoute } from '@tanstack/react-router'
import { getAdminAgent } from '../../../api/admin'
import { AdminAgentGuardrailEditPage } from '../../../features/admin-agents/AdminAgentGuardrailEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/agents_/$agentId/guardrail/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const agent = await getAdminAgent(params.agentId)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username, agent }
  },
  component: () => (
    <PageMarker kind="admin-agent-guardrail-edit">
      <AdminAgentGuardrailEditPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
