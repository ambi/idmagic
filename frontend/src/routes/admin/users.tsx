import { createFileRoute } from '@tanstack/react-router'
import { listAdminUsersPage } from '../../api'
import { AdminUsersPage } from '../../features/admin-users/AdminUsersListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/users')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const page = await listAdminUsersPage()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      users: page.users,
      nextCursor: page.nextCursor,
    }
  },
  component: AdminUsersRoute,
})

function AdminUsersRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-users">
      <AdminUsersPage {...data} />
    </PageMarker>
  )
}
