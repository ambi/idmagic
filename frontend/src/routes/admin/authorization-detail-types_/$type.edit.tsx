import { createFileRoute } from '@tanstack/react-router'
import { listAuthorizationDetailTypes } from '../../../api/admin'
import { AuthenticationAPIError } from '../../../api/core'
import { AdminAuthorizationDetailTypeEditPage } from '../../../features/admin-authz-detail-types/AdminAuthorizationDetailTypeEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/authorization-detail-types_/$type/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const detailType = (await listAuthorizationDetailTypes()).find(
      (item) => item.type === params.type,
    )
    if (!detailType) {
      throw new AuthenticationAPIError('Authorization detail type not found.', 'not_found')
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      detailType,
    }
  },
  component: AdminAuthorizationDetailTypeEditRoute,
})

function AdminAuthorizationDetailTypeEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-authz-detail-type-edit">
      <AdminAuthorizationDetailTypeEditPage {...data} />
    </PageMarker>
  )
}
