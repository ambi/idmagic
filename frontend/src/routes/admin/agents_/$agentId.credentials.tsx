import { createFileRoute } from '@tanstack/react-router'
import { getAdminAgent } from '../../../api/admin'
import { AdminAgentCredentialsPage } from '../../../features/admin-agents/AdminAgentCredentialsPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/agents_/$agentId/credentials')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const agent = await getAdminAgent(params.agentId)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username, agent }
  },
  component: () => (
    <PageMarker kind="admin-agent-credentials">
      <AdminAgentCredentialsPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
