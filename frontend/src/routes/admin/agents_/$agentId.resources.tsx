import { createFileRoute } from '@tanstack/react-router'
import { AdminAgentResourcesPage } from '../../../features/admin-agents/AdminAgentResourcesPage'
import { PageMarker } from '../../-page'
import { loadAgentResources } from './-resources'

export const Route = createFileRoute('/admin/agents_/$agentId/resources')({
  loader: ({ location, params }) =>
    loadAgentResources(location.pathname, location.searchStr, params.agentId),
  component: () => (
    <PageMarker kind="admin-agent-resources">
      <AdminAgentResourcesPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
