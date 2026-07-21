import { createFileRoute } from '@tanstack/react-router'
import { getAdminGroup } from '../../../api/admin'
import { request } from '../../../api/core'
import { AdminGroupEditPage } from '../../../features/admin-groups/AdminGroupEditPage'
import type { TenantUserAttributeSchema } from '../../../types'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/$groupId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [{ group }, schema] = await Promise.all([
      getAdminGroup(params.groupId),
      request<TenantUserAttributeSchema>('/api/admin/tenant/user_attribute_schema'),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      group,
      schema,
    }
  },
  component: AdminGroupEditRoute,
})

function AdminGroupEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-edit">
      <AdminGroupEditPage {...data} />
    </PageMarker>
  )
}
