import { createFileRoute } from '@tanstack/react-router'
import { AdminUserCreatePage } from '../../../features/admin-users/AdminUsersPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/users_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminUserCreateRoute,
})

function AdminUserCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-user-create">
      <AdminUserCreatePage {...data} />
    </PageMarker>
  )
}
