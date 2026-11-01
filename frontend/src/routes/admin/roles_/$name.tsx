import { createFileRoute } from '@tanstack/react-router'
import { AuthenticationAPIError, request } from '../../../api/core'
import { listAdminRoleUsers } from '../../../api/admin'
import { AdminRoleDetailPage } from '../../../features/admin-roles/AdminRolesPage'
import type { AdminRole } from '../../../types'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

type AdminRoleListResponse = { roles: AdminRole[] }

export const Route = createFileRoute('/admin/roles_/$name')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [roles, users] = await Promise.all([
      request<AdminRoleListResponse>('/api/admin/v1/policy/roles'),
      listAdminRoleUsers(),
    ])
    const role = roles.roles.find((r) => r.name === params.name)
    if (!role) throw new AuthenticationAPIError('Role not found.', 'not_found')
    const usernames = users
      .filter((u) => u.roles.includes(params.name))
      .map((u) => u.preferred_username)
    return {
      actorUsername: account.preferred_username,
      role,
      count: usernames.length,
      usernames,
    }
  },
  component: AdminRoleDetailRoute,
})

function AdminRoleDetailRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-role-detail">
      <AdminRoleDetailPage {...data} />
    </PageMarker>
  )
}
