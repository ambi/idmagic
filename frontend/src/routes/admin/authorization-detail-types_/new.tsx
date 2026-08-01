import { createFileRoute } from '@tanstack/react-router'
import { AdminAuthorizationDetailTypeCreatePage } from '../../../features/admin-authz-detail-types/AdminAuthorizationDetailTypeCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/authorization-detail-types_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminAuthorizationDetailTypeCreateRoute,
})

function AdminAuthorizationDetailTypeCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-authz-detail-type-create">
      <AdminAuthorizationDetailTypeCreatePage {...data} />
    </PageMarker>
  )
}
