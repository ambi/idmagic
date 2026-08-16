import { createFileRoute } from '@tanstack/react-router'
import { listWorkloadTrustBundles } from '../../api/admin'
import { AdminWorkloadTrustBundlesPage } from '../../features/admin-workload-identity/AdminWorkloadTrustBundlesPage'
import { loadRejectionEvents } from '../../features/admin-workload-identity/rejections'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/workload-identity')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [trustBundles, rejections] = await Promise.all([
      listWorkloadTrustBundles(),
      loadRejectionEvents(),
    ])
    return {
      actorUsername: account.preferred_username,
      trustBundles,
      rejectionEvents: rejections.events,
      rejectionsUnavailable: rejections.unavailable,
      rejectionsTruncated: rejections.truncated,
    }
  },
  component: AdminWorkloadTrustBundlesRoute,
})

function AdminWorkloadTrustBundlesRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-workload-identity">
      <AdminWorkloadTrustBundlesPage {...data} />
    </PageMarker>
  )
}
