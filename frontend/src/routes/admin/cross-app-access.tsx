import { createFileRoute } from '@tanstack/react-router'
import { listCrossAppAccessIssuePolicies, listCrossAppAccessRedeemPolicies } from '../../api/admin'
import { AdminCrossAppAccessPage } from '../../features/admin-cross-app-access/AdminCrossAppAccessPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/cross-app-access')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [issuePolicies, redeemPolicies] = await Promise.all([
      listCrossAppAccessIssuePolicies(),
      listCrossAppAccessRedeemPolicies(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      issuePolicies,
      redeemPolicies,
    }
  },
  component: AdminCrossAppAccessRoute,
})

function AdminCrossAppAccessRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-cross-app-access">
      <AdminCrossAppAccessPage {...data} />
    </PageMarker>
  )
}
