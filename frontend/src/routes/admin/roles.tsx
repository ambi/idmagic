import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../api/core'
import { listAdminRoleUsers } from '../../api/admin'
import { AdminRolesPage } from '../../features/admin-roles/AdminRolesPage'
import type { AdminRole } from '../../types'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

type AdminRoleListResponse = { roles: AdminRole[] }

export const Route = createFileRoute('/admin/roles')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [roles, users] = await Promise.all([
      request<AdminRoleListResponse>('/api/admin/v1/policy/roles'),
      listAdminRoleUsers(),
    ])
    return {
      actorUsername: account.preferred_username,
      roles: roles.roles,
      users,
    }
  },
  component: AdminRolesRoute,
})

function AdminRolesRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-roles">
      <AdminRolesPage {...data} />
    </PageMarker>
  )
}
