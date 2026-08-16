import { createFileRoute } from '@tanstack/react-router'
import { AdminWorkloadTrustBundleCreatePage } from '../../../features/admin-workload-identity/AdminWorkloadTrustBundleCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/workload-identity_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminWorkloadTrustBundleCreateRoute,
})

function AdminWorkloadTrustBundleCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-workload-trust-bundle-create">
      <AdminWorkloadTrustBundleCreatePage {...data} />
    </PageMarker>
  )
}
