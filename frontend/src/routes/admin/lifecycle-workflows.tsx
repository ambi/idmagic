import { createFileRoute } from '@tanstack/react-router'
import { listLifecycleWorkflows } from '../../api/admin'
import { AdminLifecycleWorkflowsPage } from '../../features/admin-lifecycle-workflows/AdminLifecycleWorkflowsPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'
export const Route = createFileRoute('/admin/lifecycle-workflows')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      workflows: await listLifecycleWorkflows(),
    }
  },
  component: () => (
    <PageMarker kind="admin-lifecycle-workflows">
      <AdminLifecycleWorkflowsPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})
