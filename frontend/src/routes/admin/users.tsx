import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../api/core'
import { AdminUsersPage } from '../../features/admin-users/AdminUsersListPage'
import type { AdminUser } from '../../types'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

type AdminUserListResponse = { users: AdminUser[] }

export const Route = createFileRoute('/admin/users')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const users = await request<AdminUserListResponse>('/api/admin/users')
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      users: users.users,
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
