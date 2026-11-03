import { createFileRoute } from '@tanstack/react-router'
import { AdminAgentResourcesPage } from '../../../features/admin-agents/AdminAgentResourcesPage'
import { PageMarker } from '../../-page'
import { loadAgentResources } from './-resources'

export const Route = createFileRoute('/admin/agents_/$agentId/resources_/edit')({
  validateSearch: (search: Record<string, unknown>) => ({
    server: typeof search.server === 'string' ? search.server : '',
  }),
  loader: ({ location, params }) =>
    loadAgentResources(location.pathname, location.searchStr, params.agentId),
  component: () => (
    <PageMarker kind="admin-agent-resources-edit">
      <AdminAgentResourcesPage
        {...Route.useLoaderData()}
        edit
        selectedServer={Route.useSearch().server}
      />
    </PageMarker>
  ),
})
