import { createFileRoute } from '@tanstack/react-router'
import { listAdminApplications, listAdminGroups } from '../../../api/admin'
import { AdminLifecycleWorkflowCreatePage } from '../../../features/admin-lifecycle-workflows/AdminLifecycleWorkflowEditorPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/lifecycle-workflows_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [groups, applications] = await Promise.all([listAdminGroups(), listAdminApplications()])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      groups,
      applications,
    }
  },
  component: AdminLifecycleWorkflowCreateRoute,
})

function AdminLifecycleWorkflowCreateRoute() {
  return (
    <PageMarker kind="admin-lifecycle-workflow-create">
      <AdminLifecycleWorkflowCreatePage {...Route.useLoaderData()} />
    </PageMarker>
  )
}
