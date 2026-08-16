import { createFileRoute } from '@tanstack/react-router'
import { getWorkloadTrustBundle } from '../../../api/admin'
import { AdminWorkloadTrustBundleEditPage } from '../../../features/admin-workload-identity/AdminWorkloadTrustBundleEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/workload-identity_/$trustBundleId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const trustBundle = await getWorkloadTrustBundle(params.trustBundleId)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      trustBundle,
    }
  },
  component: AdminWorkloadTrustBundleEditRoute,
})

function AdminWorkloadTrustBundleEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-workload-trust-bundle-edit">
      <AdminWorkloadTrustBundleEditPage {...data} />
    </PageMarker>
  )
}
