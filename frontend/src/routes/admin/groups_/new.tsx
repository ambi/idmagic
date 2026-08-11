import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../../api/core'
import { AdminGroupCreatePage } from '../../../features/admin-groups/AdminGroupCreatePage'
import type { TenantGroupAttributeSchema } from '../../../types'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const groupAttributeSchema = await request<TenantGroupAttributeSchema>(
      '/api/admin/v1/tenant/group_attribute_schema',
    )
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      groupAttributeSchema,
    }
  },
  component: AdminGroupCreateRoute,
})

function AdminGroupCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-create">
      <AdminGroupCreatePage {...data} />
    </PageMarker>
  )
}
