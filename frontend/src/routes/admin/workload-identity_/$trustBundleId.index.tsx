import { createFileRoute } from '@tanstack/react-router'
import {
  getWorkloadTrustBundle,
  listAdminAgents,
  listAgentWorkloadBindings,
} from '../../../api/admin'
import { AdminWorkloadTrustBundleDetailPage } from '../../../features/admin-workload-identity/AdminWorkloadTrustBundleDetailPage'
import { loadRejectionEvents } from '../../../features/admin-workload-identity/rejections'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/workload-identity_/$trustBundleId/')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    // agents は subject_pattern の対応先を選ぶ picker と、agent_id を名前へ解決するために読む。
    const [trustBundle, bindings, agents, rejections] = await Promise.all([
      getWorkloadTrustBundle(params.trustBundleId),
      listAgentWorkloadBindings(params.trustBundleId),
      listAdminAgents(),
      loadRejectionEvents(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      trustBundle,
      bindings,
      agents,
      rejectionEvents: rejections.events,
      rejectionsUnavailable: rejections.unavailable,
      rejectionsTruncated: rejections.truncated,
    }
  },
  component: AdminWorkloadTrustBundleDetailRoute,
})

function AdminWorkloadTrustBundleDetailRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-workload-trust-bundle-detail">
      <AdminWorkloadTrustBundleDetailPage {...data} />
    </PageMarker>
  )
}
