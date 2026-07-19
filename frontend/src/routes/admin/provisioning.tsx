import { createFileRoute } from '@tanstack/react-router'
import { listAdminApplications, listAdminTenantProvisioningConnections } from '../../api'
import { AdminProvisioningOverviewPage } from '../../features/admin-provisioning/AdminProvisioningOverviewPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/provisioning')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [connections, applications] = await Promise.all([
      listAdminTenantProvisioningConnections(),
      listAdminApplications(),
    ])
    return {
      actorUsername: account.preferred_username,
      connections,
      applications,
    }
  },
  component: AdminProvisioningOverviewRoute,
})

function AdminProvisioningOverviewRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-provisioning">
      <AdminProvisioningOverviewPage {...data} />
    </PageMarker>
  )
}
