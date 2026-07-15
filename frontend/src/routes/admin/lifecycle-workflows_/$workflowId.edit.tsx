import { createFileRoute } from '@tanstack/react-router'
import { getLifecycleWorkflow, listAdminApplications, listAdminGroups } from '../../../api/admin'
import { AdminLifecycleWorkflowEditPage } from '../../../features/admin-lifecycle-workflows/AdminLifecycleWorkflowEditorPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/lifecycle-workflows_/$workflowId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [initialWorkflow, groups, applications] = await Promise.all([
      getLifecycleWorkflow(params.workflowId),
      listAdminGroups(),
      listAdminApplications(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      initialWorkflow,
      groups,
      applications,
    }
  },
  component: AdminLifecycleWorkflowEditRoute,
})

function AdminLifecycleWorkflowEditRoute() {
  return (
    <PageMarker kind="admin-lifecycle-workflow-edit">
      <AdminLifecycleWorkflowEditPage {...Route.useLoaderData()} />
    </PageMarker>
  )
}
